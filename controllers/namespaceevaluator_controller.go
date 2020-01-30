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
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// NamespaceEvaluatorReconciler reconciles a NamespaceEvaluator object
type NamespaceEvaluatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=namespaceevaluators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=namespaceevaluators/status,verbs=get;update;patch

func (r *NamespaceEvaluatorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile").WithValues("namespace", req.Name)
	l.Info("Reconciling")

	evaluator := watcherv1.NamespaceEvaluator{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: watcherv1.NamespaceEvaluatorMetadataName}, &evaluator); err != nil {
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

	namespace := corev1.Namespace{}
	if err := r.Get(ctx, req.NamespacedName, &namespace); err != nil && !apierrs.IsNotFound(err) {
		l.Error(err, "unable to fetch namespace")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var resourceRules rules.ResourceRules = evaluator.Spec.Rules
	evaluationResult := resourceRules.EvaluateAll(ctx, req, r.Client, l, namespace)
	if evaluationResult.Err != nil {
		l.Error(evaluationResult.Err, "hit error with evaluation")
		return ctrl.Result{}, evaluationResult.Err
	}
	annotations := namespace.GetAnnotations()
	annotations[AnnotationCheckedTime] = time.Now().Format(time.RFC3339)
	annotations[AnnotationIssue] = evaluationResult.IssuesLabelsAsString()
	namespace.SetAnnotations(annotations)
	if err := r.Update(ctx, &namespace); err != nil {
		l.Error(err, "unable to update annotations")
	}

	if annotations[AnnotationIssue] != "" {
		msg := evaluationResult.IssueMessagesAsString()
		l.Info(msg)
		if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
			l.Error(err, "Failed to send message to slack", "msg", msg)
		}
	}

	return ctrl.Result{}, nil
}

func (r *NamespaceEvaluatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log
	if err := mgr.GetFieldIndexer().IndexField(&corev1.Namespace{}, ".metadata.name", func(rawObj runtime.Object) []string {
		namespace := rawObj.(*corev1.Namespace)
		l.Info("indexing", "namespace", namespace.Name)
		return []string{namespace.Name}
	}); err != nil {
		return err
	}
	l.Info("init manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&watcherv1.NamespaceEvaluator{}).
		For(&corev1.Namespace{}).
		WithEventFilter(GetPredicateFuncs(l)).
		Complete(r)
}
