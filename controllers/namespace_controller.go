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
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"sync"

	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	Reconciler
}

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=merlin.mercari.com,resources=namespaces/status,verbs=get;update;patch

func (r *NamespaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile")

	//  check if it's clusterRule or rule changes
	resourceNames := strings.Split(req.Name, Separator)
	if len(resourceNames) >= 2 {
		l = l.WithValues("rule", req.Name)
		var rule merlinv1.Rule
		var err error
		switch resourceNames[0] {
		case GetStructName(merlinv1.ClusterRuleNamespaceRequiredLabel{}):
			rule = &merlinv1.ClusterRuleNamespaceRequiredLabel{}
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: resourceNames[1]}, rule); err != nil && !apierrs.IsNotFound(err) {
				return ctrl.Result{RequeueAfter: RequeueIntervalForError}, err
			}

		default:
			// this should not happen since we only watches resources we care, but just in case we forget to add handling
			e := fmt.Errorf("unexpected resource change")
			l.Error(e, req.NamespacedName.String())
			return ctrl.Result{}, e
		}
		if apierrs.IsNotFound(err) {
			// TODO: resource is deleted, clear all alerts
			return ctrl.Result{}, nil
		}
		if _, ok := r.RuleStatues[rule.GetName()]; !ok {
			r.RuleStatues[rule.GetName()] = &RuleStatusWithLock{}
		}
		r.RuleStatues[rule.GetName()].Lock()
		if err := rule.Evaluate(ctx, r.Client, l, nil, r.Notifiers); err != nil {
			return ctrl.Result{}, err
		}
		r.Generations.Store(rule.GetName(), rule.GetGeneration()+1)
		r.RuleStatues[rule.GetName()].RuleStatus = rule.GetStatus()
		r.RuleStatues[rule.GetName()].Unlock()
		return ctrl.Result{}, nil
	}

	l = l.WithValues("namespace", req.Name)
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
	for _, rule := range rulesToApply {
		if _, ok := r.RuleStatues[rule.GetName()]; !ok {
			r.RuleStatues[rule.GetName()] = &RuleStatusWithLock{}
		}
		r.RuleStatues[rule.GetName()].Lock()
		if err := rule.Evaluate(ctx, r.Client, l, namespace, r.Notifiers); err != nil {
			return ctrl.Result{}, err
		}
		r.Generations.Store(rule.GetName(), rule.GetGeneration()+1)
		r.RuleStatues[rule.GetName()].RuleStatus = rule.GetStatus()
		r.RuleStatues[rule.GetName()].Unlock()
	}

	return ctrl.Result{}, nil
}

func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log.WithName("SetupWithManager")
	r.Generations = &sync.Map{}
	r.RuleStatues = map[string]*RuleStatusWithLock{}

	if err := mgr.GetFieldIndexer().IndexField(&merlinv1.ClusterRuleNamespaceRequiredLabel{}, indexField, func(rawObj runtime.Object) []string {
		obj := rawObj.(*merlinv1.ClusterRuleNamespaceRequiredLabel)
		l.Info("indexing", GetStructName(obj), obj.Name)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(&corev1.Namespace{}, ".metadata.name", func(rawObj runtime.Object) []string {
		namespace := rawObj.(*corev1.Namespace)
		l.Info("indexing", "namespace", namespace.Name)
		return []string{namespace.Name}
	}); err != nil {
		return err
	}

	l.Info("initialize manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Watches(
			&source.Kind{Type: &merlinv1.ClusterRuleNamespaceRequiredLabel{}},
			&EventHandler{Log: l, Kind: GetStructName(merlinv1.ClusterRuleNamespaceRequiredLabel{}), ObjectGenerations: r.Generations}).
		WithEventFilter(GetPredicateFuncs(l, &sync.Map{})).
		Named(corev1.Namespace{}.Kind).
		Complete(r)
}

func (r *NamespaceReconciler) ListRules(ctx context.Context, req ctrl.Request, namespace corev1.Namespace) ([]merlinv1.Rule, error) {
	l := r.Log.WithName("ListRules").WithValues("namespace", req.Namespace)
	var rules []merlinv1.Rule
	requiredLabels := merlinv1.ClusterRuleNamespaceRequiredLabelList{}
	if err := r.List(ctx, &requiredLabels); client.IgnoreNotFound(err) != nil {
		l.Error(err, "failed to get ClusterRuleNamespaceRequiredLabel")
		return rules, err
	}

	for _, cRule := range requiredLabels.Items {
		ignoreNamespace := false
		for _, ns := range cRule.Spec.IgnoreNamespaces {
			if ns == req.Name { // note for namespace resource, its "namespace" is empty string
				ignoreNamespace = true
			}
		}
		if !ignoreNamespace {
			rules = append(rules, &cRule)
		}
	}
	return rules, nil
}
