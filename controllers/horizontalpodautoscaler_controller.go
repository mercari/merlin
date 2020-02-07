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
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

// HorizontalPodAutoscalerReconciler reconciles a HorizontalPodAutoscaler object
type HorizontalPodAutoscalerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=merlin.mercari.com,resources=horizontalpodautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=autoscalingv1,resources=hpa,verbs=get;list;watch

func (r *HorizontalPodAutoscalerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile").WithValues("namespace", req.Namespace, "HPA", req.Name)

	hpa := autoscalingv1.HorizontalPodAutoscaler{}
	if err := r.Client.Get(ctx, req.NamespacedName, &hpa); client.IgnoreNotFound(err) != nil {
		l.Error(err, "failed to get hpa")
		return ctrl.Result{}, err
	}

	// get list of applicable rules
	rulesToApply, err := r.ListRules(ctx, req, hpa)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(rulesToApply) == 0 {
		l.Info("No rules found to apply")
		return ctrl.Result{}, nil
	}

	// running evaluation and combine results
	l.Info("Evaluating HPA")
	evaluationResult := &merlinv1.EvaluationResult{NamespacedName: req.NamespacedName}
	for _, rule := range rulesToApply {
		evaluationResult.Combine(rule.Evaluate(ctx, r.Client, l, hpa))
	}
	l.Info("results", "issues", evaluationResult.String())

	// update annotations
	annotations := hpa.GetAnnotations()
	annotations[AnnotationCheckedTime] = time.Now().Format(time.RFC3339)
	annotations[AnnotationIssue] = evaluationResult.IssuesLabelsAsString()
	hpa.SetAnnotations(annotations)
	if err := r.Update(ctx, &hpa); err != nil {
		l.Error(err, "unable to update annotations")
	}

	// send messages if there's any issues
	if annotations[AnnotationIssue] != "" {
		msg := evaluationResult.String()
		l.Info(msg)
		notifierList := merlinv1.NotifierList{}
		if err := r.List(ctx, &notifierList); client.IgnoreNotFound(err) != nil {
			l.Error(err, "failed to get NotifierList")
			return ctrl.Result{}, err
		}
		notifierList.NotifyAll(*evaluationResult, l)
	}

	return ctrl.Result{}, nil
}

func (r *HorizontalPodAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log
	if err := mgr.GetFieldIndexer().IndexField(&autoscalingv1.HorizontalPodAutoscaler{}, ".metadata.name", func(rawObj runtime.Object) []string {
		hpa := rawObj.(*autoscalingv1.HorizontalPodAutoscaler)
		l.Info("indexing", "hpa", hpa.Name)
		return []string{hpa.Name}
	}); err != nil {
		return err
	}
	l.Info("init manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&merlinv1.ClusterRuleHPAInvalidScaleTargetRef{}).
		For(&merlinv1.ClusterRuleHPAReplicaPercentage{}).
		For(&merlinv1.RuleHPAReplicaPercentage{}).
		For(&autoscalingv1.HorizontalPodAutoscaler{}).
		WithEventFilter(GetPredicateFuncs(l)).
		Complete(r)
}

func (r *HorizontalPodAutoscalerReconciler) ListRules(ctx context.Context, req ctrl.Request, hpa autoscalingv1.HorizontalPodAutoscaler) ([]Rule, error) {
	l := r.Log.WithName("ListRules").WithValues("namespace", req.Namespace, "name", req.Name)
	var rulesToApply []Rule
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
			rulesToApply = append(rulesToApply, cRule)
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
				rulesToApply = append(rulesToApply, r)
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
				rulesToApply = append(rulesToApply, cRule)
			}
		}
	}
	return rulesToApply, nil
}
