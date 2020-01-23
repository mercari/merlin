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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

type DeploymentIssue = string

const (
	DeploymentEvaluatorAnnotationCheckedTime = "deploymentevaluator.watcher.merlin.mercari.com/checked-at"
	DeploymentEvaluatorAnnotationIssue       = "deploymentevaluator.watcher.merlin.mercari.com/issue"
)

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
	if lastChecked, ok := deployment.Annotations[DeploymentEvaluatorAnnotationCheckedTime]; ok {
		lastCheckedTime, err := time.Parse(time.RFC3339, lastChecked)
		if err != nil {
			return ctrl.Result{}, err
		}

		if lastCheckedTime.Add(3 * time.Second).After(time.Now()) {
			l.Info("last check within 3 sec, will skip")
			return ctrl.Result{}, err
		}
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

	deployment.SetAnnotations(map[string]string{
		DeploymentEvaluatorAnnotationCheckedTime: time.Now().Format(time.RFC3339),
		DeploymentEvaluatorAnnotationIssue:       "",
	})

	evaluationResult := evaluator.Spec.Rules.Evaluate(ctx, req, r.Client, deployment)
	if evaluationResult.Err != nil {
		l.Error(evaluationResult.Err, "hit error with evaluation")
		return ctrl.Result{}, evaluationResult.Err
	}
	if len(evaluationResult.Issues) > 0 {
		l.Info("deployment has issues", "issues", evaluationResult.Issues, "deployment", deployment.Name)
		deployment.Annotations[DeploymentEvaluatorAnnotationIssue] = evaluationResult.Issues.String()
	}

	if err := r.Update(ctx, &deployment); err != nil {
		l.Error(err, "unable to update deployment annotations")
	}

	// other checks for deployments
	return ctrl.Result{}, nil
}

func (r *DeploymentEvaluatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log.WithName("Setup")
	if err := mgr.GetFieldIndexer().IndexField(&appsv1.Deployment{}, ".metadata.name", func(rawObj runtime.Object) []string {
		deployment := rawObj.(*appsv1.Deployment)
		l.Info("indexing", "deployment", deployment.Name)
		return []string{deployment.Name}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&watcherv1.DeploymentEvaluator{}).
		For(&appsv1.Deployment{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			UpdateFunc:  func(e event.UpdateEvent) bool { return true },
			GenericFunc: func(e event.GenericEvent) bool { return true },
		}).
		Complete(r)
}
