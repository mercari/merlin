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
	watcherv1 "github.com/kouzoh/merlin/api/v1"
	"github.com/kouzoh/merlin/rules"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type HPAEvaluatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=hpaevaluators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=hpaevaluators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=autoscalingv1,resources=hpa,verbs=get;list;watch

func (r *HPAEvaluatorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile").WithValues("namespace", req.Namespace, "HPA", req.Name)
	l.Info("Starting reconcile")

	evaluator := watcherv1.HPAEvaluator{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: watcherv1.HPAEvaluatorMetadataName}, &evaluator); err != nil {
		l.Error(err, "failed to get evaluator")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if evaluator.IsNamespaceIgnored(req.Namespace) {
		return ctrl.Result{}, nil
	}

	notifiers := watcherv1.Notifiers{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: watcherv1.NotifiersMetadataName}, &notifiers); err != nil {
		l.Error(err, "failed to get notifier")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	hpa := autoscalingv1.HorizontalPodAutoscaler{}
	if err := r.Client.Get(ctx, req.NamespacedName, &hpa); err != nil {
		l.Error(err, "failed to get hpa")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var resourceRules rules.ResourceRules = evaluator.Spec.Rules
	evaluationResult := resourceRules.EvaluateAll(ctx, req, r.Client, l, hpa)
	if evaluationResult.Err != nil {
		l.Error(evaluationResult.Err, "hit error with evaluation")
		return ctrl.Result{}, evaluationResult.Err
	}

	hpa.SetAnnotations(map[string]string{
		AnnotationCheckedTime: time.Now().Format(time.RFC3339),
		AnnotationIssue:       evaluationResult.Issues.String(),
	})

	if err := r.Update(ctx, &hpa); err != nil {
		l.Error(err, "unable to update hpa annotations")
	}

	return ctrl.Result{}, nil
}

func (r *HPAEvaluatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
		For(&watcherv1.HPAEvaluator{}).
		For(&autoscalingv1.HorizontalPodAutoscaler{}).
		WithEventFilter(GetPredicateFuncs(l)).
		Complete(r)
}
