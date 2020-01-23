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
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	hpas := autoscalingv1.HorizontalPodAutoscalerList{}
	if err := r.List(ctx, &hpas, &client.ListOptions{Namespace: req.Namespace}); err != nil && !apierrs.IsNotFound(err) {
		l.Error(err, "unable to fetch hpas")
		return ctrl.Result{}, err
	}
	for _, hpa := range hpas.Items {
		if hpa.Spec.MaxReplicas == hpa.Status.CurrentReplicas {
			msg := "HPA Current replicas are equal to Max replicas"
			l.Info(msg)
			if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
				l.Error(err, "Failed to send message to slack")
			}
		}
		if hpa.Status.CurrentCPUUtilizationPercentage == nil {
			msg := "HPA config is not setup properly"
			l.Info(msg)
			if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
				l.Error(err, "Failed to send message to slack")
			}
		}
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
