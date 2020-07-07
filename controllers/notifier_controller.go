package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/kouzoh/merlin/alert"
	merlinv1beta1 "github.com/kouzoh/merlin/api/v1beta1"
	"github.com/kouzoh/merlin/notifiers"
)

// NotifierReconciler reconciles a Notifier object
type NotifierReconciler struct {
	client.Client
	log    logr.Logger
	scheme *runtime.Scheme
	// notifierCache stores the notifiers as cache, this will be updated when any notifier updates happen,
	// and also servers as cache so we dont need to get list of notifiers every time
	cache *notifiers.Cache
	// httpClient is the client for notifier to send alerts to external systems
	httpClient *http.Client
}

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=notifiers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=merlin.mercari.com,resources=notifiers/status,verbs=get;update;patch

func (r *NotifierReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.log.WithName("Reconcile").WithValues("namespace", req.Namespace, "name", req.Name)

	notifierObject := merlinv1beta1.Notifier{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: "", Name: req.Name}, &notifierObject); err != nil {
		if apierrs.IsNotFound(err) {
			if _, ok := r.cache.Notifiers[req.Name]; ok {
				l.Info("Clear alerts from since notifier is being deleted")
				r.cache.Notifiers[req.Name].ClearAllAlerts("recover alert since notifier is being deleted")
				r.cache.Notifiers[req.Name].Notify()
				delete(r.cache.Notifiers, req.Name)
			}
			return ctrl.Result{}, nil
		}
		l.Error(err, "failed to get notifier")
		return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
	}

	_, ok := r.cache.Notifiers[req.Name]
	if !ok { // new notifier is created, just add to cache and waits for next iteration to send notifications.
		l.Info("Manager restarted or new notifier is created", "notifier", req.Name, "status", notifierObject.Status)
		if notifierObject.Status.Alerts == nil {
			notifierObject.Status = merlinv1beta1.NotifierStatus{Alerts: map[string]alert.Alert{}}
		}
		r.cache.Notifiers[req.Name] = &notifiers.Notifier{Resource: &notifierObject, Client: r.httpClient}
		r.cache.IsReady = true
		return ctrl.Result{RequeueAfter: time.Second * time.Duration(notifierObject.Spec.NotifyInterval)}, nil
	}

	l.V(1).Info("Notifier Status", "alerts", r.cache.Notifiers[req.Name].Resource.Status)
	r.cache.Notifiers[req.Name].Notify()
	notifierObject.Status = r.cache.Notifiers[req.Name].Resource.Status

	if err := r.Status().Update(ctx, &notifierObject); err != nil {
		l.Error(err, "unable to update status")
		return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
	}
	return ctrl.Result{RequeueAfter: time.Second * time.Duration(notifierObject.Spec.NotifyInterval)}, nil
}

func (r *NotifierReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.log.WithName("SetupWithManager")
	r.cache = &notifiers.Cache{Notifiers: map[string]*notifiers.Notifier{}}
	l.Info("initialize manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&merlinv1beta1.Notifier{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
