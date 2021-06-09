package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/mercari/merlin/rules"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceReconciler struct {
	client.Client
	log    logr.Logger
	scheme *runtime.Scheme
	// notifiers stores the notifiers as cache, this will be updated when any notifiers updates happen,
	// and also serves as cache so we dont need to get list of notifiers every time
	notifiers *notifiersCache
	// rules is the list of rules cached to apply for this reconciler
	rules []*rulesCache
	// resource is the kubernetes resource type that the controller watches.
	resource runtime.Object
}

func (r *ResourceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.log.WithName("ResourceReconciler").WithValues("resource", req.NamespacedName)
	if !r.notifiers.isReady {
		l.V(1).Info("Notifiers are not ready, requeue the request")
		return ctrl.Result{RequeueAfter: requeueMinInternalSeconds * time.Second}, nil
	}

	l.Info("reconciling")
	object := r.resource.DeepCopyObject()
	if err := r.Get(ctx, req.NamespacedName, object); err != nil {
		if apierrs.IsNotFound(err) {
			msg := "recover alert since resource is deleted"
			l.Info(msg)
			r.notifiers.ClearResourceAlerts(req.NamespacedName.String(), msg)
			return ctrl.Result{}, nil
		}
		l.Error(err, "unable to retrieve the object")
		return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
	}

	var rulesToApply []rules.Rule
	for _, ruleCache := range r.rules {
		if namespaceRules, ok := ruleCache.LoadNamespaced(req.Namespace); ok {
			for _, namespaceRule := range namespaceRules {
				rulesToApply = append(rulesToApply, namespaceRule)
			}
		} else if clusterRules, ok := ruleCache.LoadNamespaced(""); ok {
			for _, clusterRule := range clusterRules {
				rulesToApply = append(rulesToApply, clusterRule)
			}
		}
	}

	allRulesAreReady := true
	for _, rule := range rulesToApply {
		if !rule.IsReady() {
			// skip the rule if it's not ready, maybe being created or updated
			l.Info("rule is not ready, skipping", "rule", rule.GetName())
			allRulesAreReady = false
			continue
		}
		l.V(1).Info("evaluating rule", "rule", rule.GetName())
		a, err := rule.Evaluate(ctx, object)
		if err != nil {
			return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
		}
		r.notifiers.SetAlert(rule, a)
	}
	if !allRulesAreReady {
		l.V(1).Info("some rules were not evaluated, requeue request")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func (r *ResourceReconciler) SetupWithManager(mgr ctrl.Manager, indexingFunc func(rawObj runtime.Object) []string) error {
	ctx := context.Background()
	l := r.log.WithName("SetupWithManager")
	l.V(1).Info("getting field indexer for resource")

	if err := mgr.
		GetFieldIndexer().
		IndexField(ctx, r.resource, metadataNameField, indexingFunc); err != nil {
		return err
	}

	l.Info("initialize manager", "rules", r.rules)
	return ctrl.
		NewControllerManagedBy(mgr).
		For(r.resource).
		WithEventFilter(&EventFilter{Log: l}).
		Named(GetStructName(r.resource)).
		Complete(r)
}
