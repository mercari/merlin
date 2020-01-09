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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strings"

	watcherv1 "github.com/kouzoh/merlin/api/v1"
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
	l := r.Log.WithName("Reconcile").WithValues("namespace", req.Namespace, "deployment name", req.Name)
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

	if evaluator.Spec.Canary.Enabled {
		hasCanaryDeployment := false
		deployments := appsv1.DeploymentList{}
		if err := r.List(ctx, &deployments, &client.ListOptions{Namespace: req.Namespace}); err != nil {
			l.Error(err, "unable to fetch Deployments")
			return ctrl.Result{}, ignoreNotFound(err)
		}
		for _, d := range deployments.Items {
			// TODO: better way to check? e.g., generic checks for canary deployment? and what if a service has multiple deployments..
			if strings.HasSuffix(d.Name, "-canary") {
				hasCanaryDeployment = true
			}
		}
		if !hasCanaryDeployment {
			msg := fmt.Sprintf("Namespace `%s` has no canary deployment", req.Namespace)
			l.Info(msg, "namespace", req.Namespace)
			if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
				l.Error(err, "Failed to send message to slack")
			}
		}
	}

	// TODO: add other checks for deployments

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
