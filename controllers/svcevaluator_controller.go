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
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	watcherv1 "github.com/kouzoh/merlin/api/v1"
)

type SVCEvaluatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=svcevaluators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=svcevaluators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=corev1,resources=svc,verbs=get;list;watch
// +kubebuilder:rbac:groups=corev1,resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups=corev1,resources=pods,verbs=get;list;watch

func (r *SVCEvaluatorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile").WithValues("namespace", req.Namespace, "SVC Name", req.Name)
	l.Info("Reconciling")

	evaluator := watcherv1.SVCEvaluator{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: watcherv1.SVCEvaluatorMetadataName}, &evaluator); err != nil {
		l.Error(err, "failed to get evaluator")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	notifiers := watcherv1.Notifiers{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: watcherv1.NotifiersMetadataName}, &notifiers); err != nil {
		l.Error(err, "failed to get notifier")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	svcs := corev1.ServiceList{}
	if err := r.List(ctx, &svcs, &client.ListOptions{Namespace: req.Namespace}); err != nil && !apierrs.IsNotFound(err) {
		l.Error(err, "unable to fetch Services")
		return ctrl.Result{}, err
	}

	for _, svc := range svcs.Items {
		lSelector, _ := v1.LabelSelectorAsSelector(v1.SetAsLabelSelector(labels.Set(svc.Spec.Selector)))
		pods := corev1.PodList{}
		if err := r.List(ctx, &pods, &client.ListOptions{
			Namespace:     req.Namespace,
			LabelSelector: lSelector,
		}); err != nil {
			l.Error(err, "unable to fetch pods")
		}
		if len(pods.Items) == 0 {
			msg := fmt.Sprintf("Service `%s` in `%s` namespace has no pods matched", svc.Name, svc.Namespace)
			l.Info(msg, "namespace", req.Namespace, "svc", svc.Name)
			if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
				l.Error(err, "Failed to send message to slack")
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *SVCEvaluatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log
	if err := mgr.GetFieldIndexer().IndexField(&corev1.Service{}, ".metadata.name", func(rawObj runtime.Object) []string {
		svc := rawObj.(*corev1.Service)
		l.Info("indexing", "service", svc.Name)
		return []string{svc.Name}
	}); err != nil {
		return err
	}
	l.Info("init manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&watcherv1.SVCEvaluator{}).
		For(&corev1.Service{}).
		WithEventFilter(GetPredicateFuncs(l)).
		Complete(r)
}
