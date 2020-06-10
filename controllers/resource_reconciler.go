package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kouzoh/merlin/notifiers"
	"github.com/kouzoh/merlin/rules"
)

type ResourceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// NotifierCache stores the notifiers as cache, this will be updated when any notifier updates happen,
	// and also serves as cache so we dont need to get list of notifiers every time
	NotifierCache *notifiers.Cache
	// Rules is the list of rules to apply for this reconciler
	Rules []rules.Rule
	// Resource is the kubernetes resource type that the controller watches.
	Resource runtime.Object
}

func (r *ResourceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile").WithValues("req", req.NamespacedName)
	if !r.NotifierCache.IsReady {
		l.V(1).Info("Notifier is not ready")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	l.Info("reconciling")
	object := r.Resource.DeepCopyObject()
	if err := r.Get(ctx, req.NamespacedName, object); err != nil {
		if apierrs.IsNotFound(err) {
			l.Info("resource is deleted, clear alerts")
			for _, notifier := range r.NotifierCache.Notifiers {
				l.V(1).Info("removing alert from notifier", "notifier", notifier.Resource.Name)
				notifier.ClearResourceAlerts(req.NamespacedName.String(), "recover alert since resource is deleted")
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, nil
		}
		l.Error(err, "unable to retrieve the object")
		return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
	}

	allRulesAreReady := true
	for _, rule := range r.Rules {
		if !rule.IsInitialized() {
			// user has not yet created rule for this yet, just continue
			l.V(1).Info("rule is not initialized, skipping", "rule", rule.GetName())
			continue
		}
		if !rule.IsReady() {
			// skip the rule if it's not ready, maybe being created or updated
			l.V(1).Info("rule is not ready, skipping", "rule", rule.GetName())
			allRulesAreReady = false
			continue
		}
		l.V(1).Info("evaluating rule", "rule", rule.GetName())
		a, err := rule.Evaluate(ctx, object)
		if err != nil {
			return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
		}
		for _, n := range rule.GetNotification().Notifiers {
			notifier, ok := r.NotifierCache.Notifiers[n]
			if !ok {
				l.Error(NotifierNotFoundErr, "notifier not found", "notifier", n)
				continue
			}
			l.V(1).Info("Setting alerts to notifier", "alert", a, "notifier", n)
			notifier.SetAlert(rule.GetName(), a)
		}
	}
	if !allRulesAreReady {
		l.V(1).Info("some rules were not evaluated, requeue request")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func (r *ResourceReconciler) SetupWithManager(mgr ctrl.Manager, indexingFunc func(rawObj runtime.Object) []string) error {
	ctx := context.Background()
	l := r.Log.WithName("SetupWithManager")
	l.V(1).Info("getting field indexer for resource")

	if err := mgr.
		GetFieldIndexer().
		IndexField(ctx, r.Resource, metadataNameField, indexingFunc); err != nil {
		return err
	}

	l.Info("initialize manager", "rules", r.Rules)
	return ctrl.
		NewControllerManagedBy(mgr).
		For(r.Resource).
		WithEventFilter(&EventFilter{Log: l}).
		Named(GetStructName(r.Resource)).
		Complete(r)
}
