package controllers

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/mercari/merlin/rules"
)

const FinalizerName = "rule.finalizers.merlin.mercari.com"

type RuleReconciler struct {
	client.Client
	log    logr.Logger
	scheme *runtime.Scheme
	// notifiers stores the notifiers as cache, this will be updated when any notifiers updates happen,
	// and also servers as cache so we dont need to get list of notifiers every time
	notifiers *notifiersCache
	// rules is the rules cache that this reconcilers will setup and evaluate.
	rules *rulesCache
	// ruleFactory generates new rule when there's any rule change/update/delete events.
	ruleFactory rules.RuleFactory
	// violationMetrics
	violationMetrics *prometheus.GaugeVec
}

func (r *RuleReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.log.WithName("RuleReconciler").WithValues("rule", req.NamespacedName)
	if !r.notifiers.isReady {
		l.V(1).Info("Notifiers are not ready, requeue the request")
		return ctrl.Result{RequeueAfter: requeueMinInternalSeconds * time.Second}, nil
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
		msg := "recover alert since rule is being deleted"
		l.Info(msg)
		r.notifiers.ClearRuleAlerts(rule.GetNotification().Notifiers, rule.GetName(), msg)
		r.rules.Delete(req.Namespace, req.Name)
		rule.RemoveFinalizer(FinalizerName)
		if err := r.Update(ctx, ruleObject); err != nil {
			l.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	alerts, err := rule.EvaluateAll(ctx)
	if err != nil {
		l.Error(err, "Error running evaluate")
		return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
	}

	for _, a := range alerts {
		l.V(1).Info("Setting alerts to notifiers", "alert", a)
		r.notifiers.SetAlert(rule, a)
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

type rulesCache struct {
	sync.Mutex
	rules map[string]map[string]rules.Rule
}

func (c *rulesCache) Load(namespace, name string) rules.Rule {
	c.Lock()
	r := c.rules[namespace][name]
	c.Unlock()
	return r
}

func (c *rulesCache) LoadNamespaced(namespace string) (map[string]rules.Rule, bool) {
	c.Lock()
	rs, ok := c.rules[namespace]
	c.Unlock()
	return rs, ok
}

func (c *rulesCache) Save(namespace, name string, rule rules.Rule) {
	c.Lock()
	if c.rules == nil {
		c.rules = map[string]map[string]rules.Rule{}
	}
	if _, ok := c.rules[namespace]; !ok {
		c.rules[namespace] = map[string]rules.Rule{}
	}
	c.rules[namespace][name] = rule
	c.Unlock()
}

func (c *rulesCache) Delete(namespace, name string) {
	c.Lock()
	if c.rules == nil {
		c.rules = map[string]map[string]rules.Rule{}
	}
	delete(c.rules[namespace], name)
	c.Unlock()
}
