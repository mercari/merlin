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
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=merlin.mercari.com,resources=namespaces/status,verbs=get;update;patch

func (r *NamespaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile").WithValues("namespace", req.Name)

	namespace := corev1.Namespace{}
	if err := r.Client.Get(ctx, req.NamespacedName, &namespace); client.IgnoreNotFound(err) != nil {
		l.Error(err, "failed to get namespace")
		return ctrl.Result{}, err
	}

	// get list of applicable rules
	rulesToApply, err := r.ListRules(ctx, req, namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(rulesToApply) == 0 {
		l.Info("No rules found to apply")
		return ctrl.Result{}, nil
	}

	// running evaluation and combine results
	l.Info("Evaluating namespace")
	evaluationResult := &merlinv1.EvaluationResult{NamespacedName: req.NamespacedName}
	for _, rule := range rulesToApply {
		evaluationResult.Combine(rule.Evaluate(ctx, r.Client, l, namespace))
	}

	// update annotations
	annotations := namespace.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[AnnotationCheckedTime] = time.Now().Format(time.RFC3339)
	annotations[AnnotationIssue] = evaluationResult.String()
	namespace.SetAnnotations(annotations)
	if err := r.Update(ctx, &namespace); err != nil {
		l.Error(err, "unable to update annotations")
	}

	// send messages if there's any issues
	if annotations[AnnotationIssue] != "" {
		l.Info("resource has issues", "issues", evaluationResult.String())
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

func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log.WithName("SetupWithManager")
	if err := mgr.GetFieldIndexer().IndexField(&corev1.Namespace{}, ".metadata.name", func(rawObj runtime.Object) []string {
		namespace := rawObj.(*corev1.Namespace)
		l.Info("indexing", "namespace", namespace.Name)
		return []string{namespace.Name}
	}); err != nil {
		return err
	}
	l.Info("init manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&merlinv1.ClusterRuleNamespaceRequiredLabel{}).
		For(&corev1.Namespace{}).
		WithEventFilter(GetPredicateFuncs(l)).
		Named(corev1.Namespace{}.Kind).
		Complete(r)
}

func (r *NamespaceReconciler) ListRules(ctx context.Context, req ctrl.Request, namespace corev1.Namespace) ([]Rule, error) {
	l := r.Log.WithName("ListRules").WithValues("namespace", req.Namespace)
	var rules []Rule
	requiredLabels := merlinv1.ClusterRuleNamespaceRequiredLabelList{}
	if err := r.List(ctx, &requiredLabels); client.IgnoreNotFound(err) != nil {
		l.Error(err, "failed to get ClusterRuleNamespaceRequiredLabel")
		return rules, err
	}

	for _, cRule := range requiredLabels.Items {
		ignoreNamespace := false
		for _, ns := range cRule.Spec.IgnoreNamespaces {
			if ns == req.Namespace {
				ignoreNamespace = true
			}
		}
		if !ignoreNamespace {
			rules = append(rules, cRule)
		}
	}
	return rules, nil
}
