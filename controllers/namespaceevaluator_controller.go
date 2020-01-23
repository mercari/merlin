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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

const (
	NamespaceEvaluatorAnnotationCheckedTime = "namespaceevaluator.watcher.merlin.mercari.com/checked-at"
	NamespaceEvaluatorAnnotationIssue       = "namespaceevaluator.watcher.merlin.mercari.com/issue"
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

	namespace := corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: req.Name}, &namespace); err != nil {
		l.Error(err, "unable to fetch namespace")
		return ctrl.Result{}, ignoreNotFound(err)
	}

	if lastChecked, ok := namespace.Annotations[NamespaceEvaluatorAnnotationCheckedTime]; ok {
		lastCheckedTime, err := time.Parse(time.RFC3339, lastChecked)
		if err != nil {
			return ctrl.Result{}, err
		}

		if lastCheckedTime.Add(3 * time.Second).After(time.Now()) {
			l.Info("last check within 3 sec, will skip")
			return ctrl.Result{}, err
		}
	}

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

	namespace.SetAnnotations(map[string]string{
		NamespaceEvaluatorAnnotationCheckedTime: time.Now().Format(time.RFC3339),
		NamespaceEvaluatorAnnotationIssue:       "",
	})

	evaluationResult := evaluator.Spec.Rules.Evaluate(ctx, req, r.Client, namespace)
	if evaluationResult.Err != nil {
		l.Error(evaluationResult.Err, "hit error with evaluation")
		return ctrl.Result{}, evaluationResult.Err
	}
	if len(evaluationResult.Issues) > 0 {
		l.Info("namespace has issues", "issues", evaluationResult.Issues, "namespace", namespace.Name)
		namespace.Annotations[NamespaceEvaluatorAnnotationIssue] = evaluationResult.Issues.String()
	}

	if err := r.Update(ctx, &namespace); err != nil {
		l.Error(err, "unable to update namespace annotations")
	}

	return ctrl.Result{}, nil
}

func (r *NamespaceEvaluatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log.WithName("Setup")
	if err := mgr.GetFieldIndexer().IndexField(&corev1.Namespace{}, ".metadata.name", func(rawObj runtime.Object) []string {
		namespace := rawObj.(*corev1.Namespace)
		l.Info("index field", "namespace", namespace.Name)
		return []string{namespace.Name}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&watcherv1.NamespaceEvaluator{}).
		For(&corev1.Namespace{}).
		WithEventFilter(predicate.Funcs{
			// While we do not care what the event contains, we should not handle Delete events or Unknown / Generic events
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			UpdateFunc:  func(e event.UpdateEvent) bool { return true },
			GenericFunc: func(e event.GenericEvent) bool { return true },
		}).
		Complete(r)
}
