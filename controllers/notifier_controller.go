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
	"github.com/go-logr/logr"
	"github.com/kouzoh/merlin/notifiers/alert"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"

	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

// NotifierReconciler reconciles a Notifier object
type NotifierReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// Notifiers stores the notifiers as cache, this will be updated when any notifier updates happen,
	// and also servers as cache so we dont need to get list of notifiers every time
	Notifiers map[string]*merlinv1.Notifier
	// Generations stores the rule generation, to be used for event filter to determine if events are from Reconciler
	// This is required since status updates also increases generation, so we cant use metadata's generation.
	Generations *sync.Map
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
			// TODO: notifier is deleted, clean up notifications
			return ctrl.Result{}, nil
		}
		l.Error(err, "failed to get notifier")
		return ctrl.Result{RequeueAfter: RequeueIntervalForError}, err
	}

	notifierCache, ok := r.Notifiers[req.Name]
	if !ok { // new notifier is created, just add to cache and waits for next iteration to send notifications.
		l.Info("Manager restarted or new notifier is created", "notifier", req.Name, "status", notifier.Status)
		if notifier.Status.Alerts == nil {
			notifier.Status = merlinv1.NotifierStatus{Alerts: map[string]alert.Alert{}}
		}
		r.Notifiers[req.Name] = &notifier
		return ctrl.Result{RequeueAfter: time.Second * time.Duration(notifier.Spec.NotifyInterval)}, nil
	}

	l.Info("Notifier Status", "alerts", notifierCache.Status)
	notifierCache.Notify(r.HttpClient)
	notifier.Status = notifierCache.Status

	r.Generations.Store(notifier.Name, notifier.Generation+1)
	if err := r.Client.Update(ctx, &notifier); err != nil {
		l.Error(err, "unable to update status")
		return ctrl.Result{RequeueAfter: RequeueIntervalForError}, err
	}
	return ctrl.Result{RequeueAfter: time.Second * time.Duration(notifier.Spec.NotifyInterval)}, nil
}

func (r *NotifierReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log.WithName("SetupWithManager")
	r.Notifiers = map[string]*merlinv1.Notifier{}
	r.Generations = &sync.Map{}
	l.Info("initialize manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&merlinv1.Notifier{}).
		WithEventFilter(GetPredicateFuncs(l, r.Generations)).
		Complete(r)
}
