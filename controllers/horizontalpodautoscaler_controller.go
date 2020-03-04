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
	"fmt"
	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"

	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

// HorizontalPodAutoscalerReconciler reconciles a HorizontalPodAutoscaler object
type HorizontalPodAutoscalerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// Notifiers stores the notifiers as cache, this will be updated when any notifier updates happen,
	// and also servers as cache so we dont need to get list of notifiers every time
	Notifiers map[string]*merlinv1.Notifier
	// Generations stores the rule generation, to be used for event filter to determine if events are from Reconciler
	// This is required since status updates also increases generation, so we cant use metadata's generation.
	Generations map[string]int64
}

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=merlin.mercari.com,resources=horizontalpodautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=autoscalingv1,resources=hpa,verbs=get;list;watch

func (r *HorizontalPodAutoscalerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile")

	// TODOL before performing evaluation for each rule, check if they're locked, if locked means there's an ongoing update to the status
	// wait for the lock (rule checking violations) to release and get the updated rule properties then check.

	//  check if it's clusterRule or rule changes
	resourceNames := strings.Split(req.Name, Separator)
	if len(resourceNames) >= 2 {
		l = l.WithValues("rule", req.NamespacedName)
		var rule merlinv1.Rule
		var err error
		switch resourceNames[0] {

		case GetStructName(merlinv1.ClusterRuleHPAInvalidScaleTargetRef{}):
			rule = &merlinv1.ClusterRuleHPAInvalidScaleTargetRef{}
			if err = r.Client.Get(ctx, types.NamespacedName{Name: resourceNames[1]}, rule); err != nil && !apierrs.IsNotFound(err) {
				return ctrl.Result{RequeueAfter: RequeueIntervalForError}, err
			}

		case GetStructName(merlinv1.ClusterRuleHPAReplicaPercentage{}):
			rule = &merlinv1.ClusterRuleHPAReplicaPercentage{}
			if err = r.Client.Get(ctx, types.NamespacedName{Name: resourceNames[1]}, rule); err != nil && !apierrs.IsNotFound(err) {
				return ctrl.Result{RequeueAfter: RequeueIntervalForError}, err
			}

		case GetStructName(merlinv1.RuleHPAReplicaPercentage{}):
			rule = &merlinv1.RuleHPAReplicaPercentage{}
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: resourceNames[1]}, rule); err != nil && !apierrs.IsNotFound(err) {
				return ctrl.Result{RequeueAfter: RequeueIntervalForError}, err
			}

		default:
			// this should not happen since we only watches resources we care, but just in case we forget to add handling
			e := fmt.Errorf("unexpected resource change")
			l.Error(e, req.NamespacedName.String())
			return ctrl.Result{}, e
		}
		if apierrs.IsNotFound(err) {
			// TODO: resource is deleted, clear all alerts
			return ctrl.Result{}, nil
		}
		if err := rule.Evaluate(ctx, r.Client, l, nil, r.Notifiers); err != nil {
			return ctrl.Result{}, err
		}
		r.Generations[rule.GetName()] = rule.GetGeneration() + 1
		return ctrl.Result{}, nil
	}
	l = l.WithValues("hpa", req.NamespacedName)
	hpa := autoscalingv1.HorizontalPodAutoscaler{}
	if err := r.Client.Get(ctx, req.NamespacedName, &hpa); client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	// update annotations
	annotations := map[string]string{}
	for k, v := range hpa.GetAnnotations() {
		if k != AnnotationCheckedTime && k != AnnotationIssue {
			annotations[k] = v
		}
	}
	hpa.SetAnnotations(annotations)
	if err := r.Update(ctx, &hpa); err != nil {
		l.Error(err, "unable to update annotations")
		return ctrl.Result{}, err
	}

	// get list of applicable rules
	rulesToApply, err := r.ListRules(ctx, req, hpa)
	if err != nil {
		return ctrl.Result{RequeueAfter: RequeueIntervalForError}, err
	}

	if len(rulesToApply) == 0 {
		l.Info("No rules found to apply")
		return ctrl.Result{}, nil
	}

	// running evaluation and combine results
	l.Info("Evaluating HPA")
	for _, rule := range rulesToApply {
		if err := rule.Evaluate(ctx, r.Client, l, hpa, r.Notifiers); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *HorizontalPodAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log.WithName("SetupWithManager")
	r.Generations = map[string]int64{}

	if err := mgr.GetFieldIndexer().IndexField(&merlinv1.ClusterRuleHPAInvalidScaleTargetRef{}, indexField, func(rawObj runtime.Object) []string {
		obj := rawObj.(*merlinv1.ClusterRuleHPAInvalidScaleTargetRef)
		l.Info("indexing", GetStructName(obj), obj.Name)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(&merlinv1.ClusterRuleHPAReplicaPercentage{}, indexField, func(rawObj runtime.Object) []string {
		obj := rawObj.(*merlinv1.ClusterRuleHPAReplicaPercentage)
		l.Info("indexing", GetStructName(obj), obj.Name)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(&merlinv1.RuleHPAReplicaPercentage{}, indexField, func(rawObj runtime.Object) []string {
		obj := rawObj.(*merlinv1.RuleHPAReplicaPercentage)
		l.Info("indexing", GetStructName(obj), obj.Name)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(&autoscalingv1.HorizontalPodAutoscaler{}, indexField, func(rawObj runtime.Object) []string {
		obj := rawObj.(*autoscalingv1.HorizontalPodAutoscaler)
		l.Info("indexing", GetStructName(obj), obj.Name)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	l.Info("initialize manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingv1.HorizontalPodAutoscaler{}).
		Watches(
			&source.Kind{Type: &merlinv1.ClusterRuleHPAInvalidScaleTargetRef{}},
			&EventHandler{Log: l, Kind: GetStructName(merlinv1.ClusterRuleHPAInvalidScaleTargetRef{}), ObjectGenerations: r.Generations}).
		Watches(
			&source.Kind{Type: &merlinv1.ClusterRuleHPAReplicaPercentage{}},
			&EventHandler{Log: l, Kind: GetStructName(merlinv1.ClusterRuleHPAReplicaPercentage{}), ObjectGenerations: r.Generations}).
		Watches(
			&source.Kind{Type: &merlinv1.RuleHPAReplicaPercentage{}},
			&EventHandler{Log: l, Kind: GetStructName(merlinv1.RuleHPAReplicaPercentage{}), ObjectGenerations: r.Generations}).
		WithEventFilter(GetPredicateFuncs(l, nil)).
		Named(autoscalingv1.HorizontalPodAutoscaler{}.Kind).
		Complete(r)
}

func (r *HorizontalPodAutoscalerReconciler) ListRules(ctx context.Context, req ctrl.Request, hpa autoscalingv1.HorizontalPodAutoscaler) ([]merlinv1.Rule, error) {
	l := r.Log.WithName("ListRules").WithValues("namespace", req.Namespace, "name", req.Name)
	var rulesToApply []merlinv1.Rule
	scaleTargetRefRules := merlinv1.ClusterRuleHPAInvalidScaleTargetRefList{}
	if err := r.List(ctx, &scaleTargetRefRules); client.IgnoreNotFound(err) != nil {
		l.Error(err, "failed to get ClusterRuleHPAInvalidScaleTargetRefList")
		return rulesToApply, err
	}

	for _, cRule := range scaleTargetRefRules.Items {
		ignoreNamespace := false
		for _, ns := range cRule.Spec.IgnoreNamespaces {
			if ns == req.Namespace {
				ignoreNamespace = true
			}
		}
		if !ignoreNamespace {
			rulesToApply = append(rulesToApply, &cRule)
		}
	}

	nsReplicaPercentageRules := merlinv1.RuleHPAReplicaPercentageList{}
	if err := r.List(ctx, &nsReplicaPercentageRules, &client.ListOptions{Namespace: req.Namespace}); client.IgnoreNotFound(err) != nil {
		l.Error(err, "failed to get RuleHPAReplicaPercentageList")
		return rulesToApply, err
	}

	// namespace rules take precedence, if there are namespace rules defined, will ignore cluster rules
	if len(nsReplicaPercentageRules.Items) > 0 {
		l.Info("Found namespace rules defined, will apply namespace rules")
		for _, r := range nsReplicaPercentageRules.Items {
			if r.Spec.Selector.Name == req.Name || r.Spec.Selector.IsLabelMatched(hpa.Labels) {
				rulesToApply = append(rulesToApply, &r)
			}
		}
	} else {
		l.Info("No namespace rules found, getting cluster rules to apply")
		replicaPercentageRulesRules := merlinv1.ClusterRuleHPAReplicaPercentageList{}
		if err := r.List(ctx, &replicaPercentageRulesRules); client.IgnoreNotFound(err) != nil {
			l.Error(err, "failed to get ClusterRuleHPAReplicaPercentageList")
			return rulesToApply, err
		}

		for _, cRule := range replicaPercentageRulesRules.Items {
			ignoreNamespace := false
			for _, ns := range cRule.Spec.IgnoreNamespaces {
				if ns == req.Namespace {
					ignoreNamespace = true
				}
			}
			if !ignoreNamespace {
				rulesToApply = append(rulesToApply, &cRule)
			}
		}
	}
	return rulesToApply, nil
}