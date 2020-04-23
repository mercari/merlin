/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/kouzoh/merlin/notifiers/alert"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

// NotifierReconciler reconciles a Notifier object
type NotifierReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// Notifiers stores the notifiers as cache, this will be updated when any notifier updates happen,
	// and also servers as cache so we dont need to get list of notifiers every time
	NotifiersCache *merlinv1.NotifiersCache
	// HttpClient is the client for notifier to send alerts to external systems
	HttpClient *http.Client
}

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=notifiers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=merlin.mercari.com,resources=notifiers/status,verbs=get;update;patch

func (r *NotifierReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile").WithValues("namespace", req.Namespace, "name", req.Name)

	notifier := merlinv1.Notifier{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: "", Name: req.Name}, &notifier); err != nil {
		if apierrs.IsNotFound(err) {
			_, ok := r.NotifiersCache.Notifiers[req.Name]
			if ok {
				l.Info("Clear alerts from since notifier is being deleted")
				r.NotifiersCache.Notifiers[req.Name].ClearAllAlerts("recover alert since notifier is being deleted")
				r.NotifiersCache.Notifiers[req.Name].Notify(r.HttpClient)
				delete(r.NotifiersCache.Notifiers, req.Name)
			}
			return ctrl.Result{}, nil
		}
		l.Error(err, "failed to get notifier")
		return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
	}

	_, ok := r.NotifiersCache.Notifiers[req.Name]
	if !ok { // new notifier is created, just add to cache and waits for next iteration to send notifications.
		l.Info("Manager restarted or new notifier is created", "notifier", req.Name, "status", notifier.Status)
		if notifier.Status.Alerts == nil {
			notifier.Status = merlinv1.NotifierStatus{Alerts: map[string]alert.Alert{}}
		}
		r.NotifiersCache.Notifiers[req.Name] = &notifier
		r.NotifiersCache.IsReady = true
		return ctrl.Result{RequeueAfter: time.Second * time.Duration(notifier.Spec.NotifyInterval)}, nil
	}

	l.V(1).Info("Notifier Status", "alerts", r.NotifiersCache.Notifiers[req.Name].Status)
	r.NotifiersCache.Notifiers[req.Name].Notify(r.HttpClient)
	notifier.Status = r.NotifiersCache.Notifiers[req.Name].Status

	if err := r.Status().Update(ctx, &notifier); err != nil {
		l.Error(err, "unable to update status")
		return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
	}
	return ctrl.Result{RequeueAfter: time.Second * time.Duration(notifier.Spec.NotifyInterval)}, nil
}

func (r *NotifierReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log.WithName("SetupWithManager")
	r.NotifiersCache = &merlinv1.NotifiersCache{
		Notifiers: map[string]*merlinv1.Notifier{},
	}
	l.Info("initialize manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&merlinv1.Notifier{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
