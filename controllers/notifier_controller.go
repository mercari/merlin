package controllers

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/mercari/merlin/alert"
	merlinv1beta1 "github.com/mercari/merlin/api/v1beta1"
	"github.com/mercari/merlin/notifiers"
	"github.com/mercari/merlin/rules"
)

type notifiersCache struct {
	sync.Mutex
	notifiers map[string]*notifiers.Notifier
	isReady   bool
}

func (n *notifiersCache) ClearResourceAlerts(resourceName, msg string) {
	n.Lock()
	for _, notifier := range n.notifiers {
		notifier.ClearResourceAlerts(resourceName, msg)
	}
	n.Unlock()
	return
}

func (n *notifiersCache) ClearRuleAlerts(ruleNotifiersNames []string, ruleName string, msg string) {
	n.Lock()
	for _, ruleNotifierName := range ruleNotifiersNames {
		if notifier, ok := n.notifiers[ruleNotifierName]; ok {
			notifier.ClearRuleAlerts(ruleName, msg)
		}
	}
	n.Unlock()
	return
}

func (n *notifiersCache) SetAlert(rule rules.Rule, a alert.Alert) {
	n.Lock()
	for _, ruleNotifierName := range rule.GetNotification().Notifiers {
		if notifier, ok := n.notifiers[ruleNotifierName]; ok {
			notifier.SetAlert(rule.GetName(), a)
		}
	}
	n.Unlock()
	return
}

// NotifierReconciler reconciles a Notifier object
type NotifierReconciler struct {
	client.Client
	log    logr.Logger
	scheme *runtime.Scheme
	// notifiers stores the notifiers as cache, this will be updated when any notifiers updates happen,
	// and also servers as cache so we dont need to get list of notifiers every time
	cache *notifiersCache
	// httpClient is the client for notifiers to send alerts to external systems
	httpClient *http.Client
	// alertMetrics is the prometheus metrics for alerts, will be 1 if the alert is firing, 0 if not.
	alertMetrics *prometheus.GaugeVec
}

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=notifiers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=merlin.mercari.com,resources=notifiers/status,verbs=get;update;patch

func (r *NotifierReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.log.WithName("NotifierReconciler").WithValues("notifier", req.Name)

	notifierObject := merlinv1beta1.Notifier{}
	if err := r.Client.Get(ctx, req.NamespacedName, &notifierObject); err != nil {
		if apierrs.IsNotFound(err) {
			if _, ok := r.cache.notifiers[req.Name]; ok {
				msg := "Clear alerts from notifier since this notifier is being deleted"
				l.Info(msg)
				r.cache.notifiers[req.Name].ClearAllAlerts(msg)
				r.cache.notifiers[req.Name].Notify()
				delete(r.cache.notifiers, req.Name)
			}
			return ctrl.Result{}, nil
		}
		l.Error(err, "failed to get notifier")
		return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
	}

	// check if notifier is cached, if not it's manager restarted or new notifier is created,
	// just add to cache and waits for next iteration to send notifications.
	if _, ok := r.cache.notifiers[req.Name]; !ok {
		l.Info("Manager restarted or new notifier is created", "status", notifierObject.Status)
		if notifierObject.Status.Alerts == nil {
			notifierObject.Status = merlinv1beta1.NotifierStatus{Alerts: map[string]alert.Alert{}}
		}
		r.cache.notifiers[req.Name] = &notifiers.Notifier{
			Resource:     &notifierObject,
			Client:       r.httpClient,
			AlertMetrics: r.alertMetrics,
		}
		r.cache.isReady = true
		return ctrl.Result{RequeueAfter: time.Second * time.Duration(notifierObject.Spec.NotifyInterval)}, nil
	}

	l.V(1).Info("Notifier Status", "alerts", r.cache.notifiers[req.Name].Resource.Status)
	r.cache.notifiers[req.Name].Notify()
	notifierObject.Status = r.cache.notifiers[req.Name].Resource.Status

	if err := r.Status().Update(ctx, &notifierObject); err != nil {
		l.Error(err, "unable to update status")
		return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
	}
	return ctrl.Result{RequeueAfter: time.Second * time.Duration(notifierObject.Spec.NotifyInterval)}, nil
}

func (r *NotifierReconciler) SetupWithManager(mgr ctrl.Manager, alertMetrics *prometheus.GaugeVec) error {
	l := r.log.WithName("SetupWithManager")
	r.cache = &notifiersCache{notifiers: map[string]*notifiers.Notifier{}}
	r.alertMetrics = alertMetrics
	l.Info("initialize manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&merlinv1beta1.Notifier{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
