package controllers

import (
	"context"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/kouzoh/merlin/notifiers"
	"github.com/kouzoh/merlin/rules"
	"github.com/prometheus/client_golang/prometheus"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const FinalizerName = "rule.finalizers.merlin.mercari.com"

type RuleReconciler struct {
	client.Client
	log    logr.Logger
	scheme *runtime.Scheme
	// notifierCache stores the notifiers as cache, this will be updated when any notifier updates happen,
	// and also servers as cache so we dont need to get list of notifiers every time
	notifierCache *notifiers.Cache
	// rules is the rules cache that this reconcilers will setup and evaluate.
	rules *rules.Cache
	// ruleFactory generates new rule when there's any rule change/update/delete events.
	ruleFactory rules.RuleFactory
	// violationMetrics
	violationMetrics *prometheus.GaugeVec
}

func (r *RuleReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.log.WithName("Reconcile").WithValues("rule", req.NamespacedName)
	if !r.notifierCache.IsReady {
		l.V(1).Info("Notifier is not ready")
		return ctrl.Result{RequeueAfter: 3 * time.Second}, nil
	}

	l.Info("Reconciling for rule changed/created")

	rule, err := r.ruleFactory.New(ctx, r.Client, l, req.NamespacedName)
	if err != nil {
		if apierrs.IsNotFound(err) {
			l.Info("rule is deleted")
			return ctrl.Result{}, nil
		}
		l.Error(err, "Failed to get rule")
		return ctrl.Result{}, err
	}
	rule.SetReady(false)
	r.rules.Save(req.Namespace, req.Name, rule)
	l.V(1).Info("rules to reconcile", "rules", r.rules)

	ruleObject := rule.GetObject()
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
			notifier, ok := r.notifierCache.Notifiers[n]
			if !ok {
				l.Error(NotifierNotFoundErr, "notifier not found", "notifier", n)
				continue
			}
			l.V(1).Info("removing alert from notifier", "notifier", notifier.Resource.Name, "ruleName", rule.GetName())
			notifier.ClearRuleAlerts(rule.GetName(), "recover alert since rule is being deleted")
			r.rules.Delete(req.Namespace, req.Name)
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
		notifier, ok := r.notifierCache.Notifiers[n]
		if !ok {
			l.Error(NotifierNotFoundErr, "notifier not found", "notifier", n)
			continue
		}
		for _, a := range alerts {
			l.V(1).Info("Setting alerts to notifier", "alert", a)
			notifier.SetAlert(rule.GetName(), a)
			resource := strings.Split(a.ResourceName, "/")
			ruleName := strings.Split(rule.GetName(), "/")
			promLabels := prometheus.Labels{
				"rule":               ruleName[0],
				"rule_name":          ruleName[1],
				"resource_namespace": resource[0],
				"resource_name":      resource[1],
				"kind":               a.ResourceKind,
			}
			if a.Violated {
				r.violationMetrics.With(promLabels).Set(1)
			} else {
				r.violationMetrics.With(promLabels).Set(0)
			}
		}
	}

	rule.SetReady(true)
	return ctrl.Result{}, nil
}

func (r *RuleReconciler) SetupWithManager(mgr ctrl.Manager, violationMetrics *prometheus.GaugeVec, clusterRule, namespaceRule runtime.Object, indexingFunc func(rawObj runtime.Object) []string) error {
	ctx := context.Background()
	l := r.log.WithName("SetupWithManager")

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
	r.violationMetrics = violationMetrics

	l.Info("initialize manager", "cluster rule", clusterRule, "namespaced rule", namespaceRule)

	return builder.
		WithEventFilter(&EventFilter{Log: l}).
		Complete(r)
}
