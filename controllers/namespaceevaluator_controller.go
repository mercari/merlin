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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strings"

	watcherv1 "github.com/kouzoh/merlin/api/v1"
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
	if err := r.Get(ctx, client.ObjectKey{Name: req.Name}, &namespace); err != nil {
		l.Error(err, "unable to fetch namespace")
		return ctrl.Result{}, ignoreNotFound(err)
	}

	if evaluator.Spec.IstioInjection.Label == watcherv1.LabelKeyExists ||
		evaluator.Spec.IstioInjection.Label == watcherv1.LabelKeyFalse ||
		evaluator.Spec.IstioInjection.Label == watcherv1.LabelKeyTrue {
		istioInjectionLabelExpected := strings.ToLower(evaluator.Spec.IstioInjection.Label)
		istioInjectionLabel, ok := namespace.Labels[watcherv1.NamespaceIstioInjecitonLabelKey]
		if !ok {
			msg := "Namespace has no istio-injection defined"
			l.Info(msg)
			if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
				l.Error(err, "Failed to send message to slack")
			}
		}

		if (istioInjectionLabelExpected == watcherv1.LabelKeyTrue || istioInjectionLabelExpected == watcherv1.LabelKeyFalse) &&
			istioInjectionLabel != istioInjectionLabelExpected {
			msg := fmt.Sprintf("Namespace's istio-injection label '%s' is different from expected '%s'", istioInjectionLabel, istioInjectionLabelExpected)
			l.Info(msg)
			if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
				l.Error(err, "Failed to send message to slack")
			}
		}
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
