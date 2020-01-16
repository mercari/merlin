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
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	watcherv1 "github.com/kouzoh/merlin/api/v1"
)

type PDBEvaluatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=pdbevaluators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=pdbevaluators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=policyv1beta1,resources=pdb,verbs=get;list;watch

func (r *PDBEvaluatorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile").WithValues("namespace", req.Namespace)
	evaluator := watcherv1.PDBEvaluator{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: watcherv1.PDBEvaluatorMetadataName}, &evaluator); err != nil {
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

	pdbs := policyv1beta1.PodDisruptionBudgetList{}
	if err := r.List(ctx, &pdbs, &client.ListOptions{Namespace: req.Namespace}); err != nil {
		l.Error(err, "unable to fetch PDBs")
		return ctrl.Result{}, ignoreNotFound(err)
	}
	for _, pdb := range pdbs.Items {
		pdbSelector := v1.SetAsLabelSelector(pdb.Spec.Selector.MatchLabels).String()
		pods := corev1.PodList{}
		if err := r.List(ctx, &pods, &client.ListOptions{
			Namespace: req.Namespace,
			Raw: &v1.ListOptions{
				LabelSelector: pdbSelector,
			},
		}); err != nil {
			l.Error(err, "unable to fetch Pods")
			return ctrl.Result{}, ignoreNotFound(err)
		}
		if len(pods.Items) == 0 {
			msg := fmt.Sprintf("PDB `%s` has no target pods", pdb.Name)
			l.Info(msg)
			if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
				l.Error(err, "Failed to send message to slack")
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *PDBEvaluatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log.WithName("Setup")
	if err := mgr.GetFieldIndexer().IndexField(&policyv1beta1.PodDisruptionBudget{}, ".metadata.name", func(rawObj runtime.Object) []string {
		pdb := rawObj.(*policyv1beta1.PodDisruptionBudget)
		l.Info("index field", "pdb", pdb.Name)
		return []string{pdb.Name}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&watcherv1.PDBEvaluator{}).
		For(&policyv1beta1.PodDisruptionBudget{}).
		WithEventFilter(predicate.Funcs{
			// While we do not care what the event contains, we should not handle Delete events or Unknown / Generic events
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			UpdateFunc:  func(e event.UpdateEvent) bool { return true },
			GenericFunc: func(e event.GenericEvent) bool { return true },
		}).
		Complete(r)
}
