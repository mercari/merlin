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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type DeploymentIssue = string

// DeploymentEvaluatorReconciler reconciles a DeploymentEvaluator object
type DeploymentEvaluatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=deploymentevaluators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=deploymentevaluators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=appsv1,resources=deployments,verbs=get;list;watch

func (r *DeploymentEvaluatorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile").WithValues("namespace", req.Namespace, "deployment", req.Name)
	deployment := appsv1.Deployment{}
	if err := r.Client.Get(ctx, req.NamespacedName, &deployment); err != nil {
		l.Error(err, "failed to get deployment")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	evaluator := watcherv1.DeploymentEvaluator{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: watcherv1.DeploymentEvaluatorMetadataName}, &evaluator); err != nil {
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

	evaluationResult := evaluator.Spec.Rules.Evaluate(ctx, req, r.Client, deployment)
	if evaluationResult.Err != nil {
		l.Error(evaluationResult.Err, "hit error with evaluation")
		return ctrl.Result{}, evaluationResult.Err
	}

	deployment.SetAnnotations(map[string]string{
		AnnotationCheckedTime: time.Now().Format(time.RFC3339),
		AnnotationIssue:       evaluationResult.Issues.String(),
	})

	if err := r.Update(ctx, &deployment); err != nil {
		l.Error(err, "unable to update deployment annotations")
	}

	// other checks for deployments
	return ctrl.Result{}, nil
}

func (r *DeploymentEvaluatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log
	if err := mgr.GetFieldIndexer().IndexField(&appsv1.Deployment{}, ".metadata.name", func(rawObj runtime.Object) []string {
		deployment := rawObj.(*appsv1.Deployment)
		l.Info("indexing", "deployment", deployment.Name)
		return []string{deployment.Name}
	}); err != nil {
		return err
	}
	l.Info("init manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&watcherv1.DeploymentEvaluator{}).
		For(&appsv1.Deployment{}).
		WithEventFilter(GetPredicateFuncs(l)).
		Complete(r)
}
