package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kouzoh/merlin/notifiers"
	"github.com/kouzoh/merlin/rules"
)

type RuleReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// NotifierCache stores the notifiers as cache, this will be updated when any notifier updates happen,
	// and also servers as cache so we dont need to get list of notifiers every time
	NotifierCache *notifiers.Cache
	// Rule is the rule that this reconcilers will setup and evaluate.
	Rule rules.Rule
}

func (r *RuleReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile").WithValues("rule", req.NamespacedName)
	if !r.NotifierCache.IsReady {
		l.V(1).Info("Notifier is not ready")
		return ctrl.Result{RequeueAfter: 3 * time.Second}, nil
	}

	l.Info("Reconciling for rule for rule changed/created")
	rule := r.Rule
	rule.SetReady(false)
	ruleObject, err := rule.GetObject(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
	}

	if rule.GetObjectMeta().DeletionTimestamp.IsZero() {
		if !containsString(rule.GetObjectMeta().Finalizers, FinalizerName) {
			l.V(1).Info("Setting finalizer", "finalizer", FinalizerName)
			rule.SetFinalizer(FinalizerName)
			if err := r.Update(ctx, ruleObject); err != nil {
				l.Error(err, "Failed to set finalizer")
				return ctrl.Result{}, err
			}
		}
	} else if containsString(rule.GetObjectMeta().Finalizers, FinalizerName) {
		l.Info("Rule is being delete, clear alerts")
		for _, n := range rule.GetNotification().Notifiers { // TODO: simplify this?
			notifier, ok := r.NotifierCache.Notifiers[n]
			if !ok {
				l.Error(NotifierNotFoundErr, "notifier not found", "notifier", n)
				continue
			}
			l.V(1).Info("removing alert from notifier", "notifier", notifier.Resource.Name)
			notifier.ClearRuleAlerts(GetStructName(ruleObject)+Separator+req.NamespacedName.String(), "recover alert since rule is being deleted")
			rule.RemoveFinalizer(FinalizerName)
			if err := r.Update(ctx, ruleObject); err != nil {
				l.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	alerts, err := rule.EvaluateAll(ctx)
	if err != nil {
		l.Error(err, "Error running evaluate")
		return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
	}

	for _, n := range rule.GetNotification().Notifiers { // TODO: simplify this?
		notifier, ok := r.NotifierCache.Notifiers[n]
		if !ok {
			l.Error(NotifierNotFoundErr, "notifier not found", "notifier", n)
			continue
		}
		for _, a := range alerts {
			l.V(1).Info("Setting alerts to notifier", "alert", a)
			notifier.SetAlert(rule.GetName(), a)
		}
	}

	rule.SetReady(true)
	return ctrl.Result{}, nil
}

func (r *RuleReconciler) SetupWithManager(mgr ctrl.Manager, clusterRule, namespaceRule runtime.Object, indexingFunc func(rawObj runtime.Object) []string) error {
	ctx := context.Background()
	l := r.Log.WithName("SetupWithManager")

	l.V(1).Info("getting field indexer for cluster rule")
	if err := mgr.
		GetFieldIndexer().
		IndexField(ctx, clusterRule, metadataNameField, indexingFunc); err != nil {
		return err
	}
	builder := ctrl.NewControllerManagedBy(mgr).For(clusterRule)
	if namespaceRule != nil {
		l.V(1).Info("getting field indexer for namespaced rule")
		if err := mgr.
			GetFieldIndexer().
			IndexField(ctx, namespaceRule, metadataNameField, indexingFunc); err != nil {
			return err
		}
		builder.Watches(&source.Kind{Type: namespaceRule}, &EventHandler{Log: l})
	}

	l.Info("initialize manager", "cluster rule", clusterRule, "namespaced rule", namespaceRule)

	return builder.
		WithEventFilter(&EventFilter{Log: l}).
		Named(GetStructName(r.Rule)).
		Complete(r)
}
