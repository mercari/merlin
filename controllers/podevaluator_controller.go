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
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	watcherv1 "github.com/kouzoh/merlin/api/v1"
)

// PodEvaluatorReconciler reconciles a PodEvaluator object
type PodEvaluatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=podevaluators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=podevaluators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=corev1,resources=services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=corev1,resources=pods,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;create;patch

func (r *PodEvaluatorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithValues("namespace", req.Namespace, "resource name", req.Name)
	notifier := watcherv1.Notifiers{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: "notifiers-sample"}, &notifier); err != nil {
		l.Error(err, "failed to get notifier")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	evaluator := watcherv1.PodEvaluator{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: "podevaluator-sample"}, &evaluator); err != nil {
		l.Error(err, "failed to get evaluator")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	pods := corev1.PodList{}
	if err := r.List(ctx, &pods, &client.ListOptions{Namespace: req.Namespace}); err != nil {
		l.Error(err, "unable to fetch Deployment")
		return ctrl.Result{}, ignoreNotFound(err)
	}
	for _, p := range pods.Items {
		// check if pod has too many restarts and not running
		for i, containerStatus := range p.Status.ContainerStatuses {
			l.Info("Checking pod",
				"notifier", notifier.Name,
				"notifier-slack url", notifier.Spec.Slack.URL,
				"name", p.Name,
				"container", p.Spec.Containers[i].Name,
				"container ready", containerStatus.Ready)
			if containerStatus.RestartCount > evaluator.Spec.Restarts && p.Status.Phase != corev1.PodRunning {
				l.Info("Pod has too many restarts and it's not running", "namespace", req.Namespace, "pod", req.Name, "restart limit", evaluator.Spec.Restarts)
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *PodEvaluatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log.WithName("SetupWithManager")
	if err := mgr.GetFieldIndexer().IndexField(&corev1.Pod{}, ".metadata.name", func(rawObj runtime.Object) []string {
		pod := rawObj.(*corev1.Pod)
		l.Info("index field", "pod", pod.Name)
		return []string{pod.Name}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&watcherv1.PodEvaluator{}).
		For(&corev1.Pod{}).
		WithEventFilter(predicate.Funcs{
			// While we do not care what the event contains, we should not handle Delete events or Unknown / Generic events
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			UpdateFunc:  func(e event.UpdateEvent) bool { return true },
			GenericFunc: func(e event.GenericEvent) bool { return true },
		}).
		Complete(r)
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}
