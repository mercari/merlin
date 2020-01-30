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
	"github.com/kouzoh/merlin/rules"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"

	watcherv1 "github.com/kouzoh/merlin/api/v1"
)

// PodEvaluatorReconciler reconciles a PodEvaluator object
type PodEvaluatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// TODO: better way of handling this - it's used to coordinate b/w reconciliation processes.
var podsInChecking = map[string]*corev1.Pod{}

// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=podevaluators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=watcher.merlin.mercari.com,resources=podevaluators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=appsv1,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=appsv1,resources=replicasets,verbs=get;list;watch
// +kubebuilder:rbac:groups=corev1,resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups=corev1,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=autoscalingv1,resources=hpa,verbs=get;list;watch
// +kubebuilder:rbac:groups=policyv1beta1,resources=pdb,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch

func (r *PodEvaluatorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("Reconcile").WithValues("namespace", req.Namespace)
	l.Info("Reconciling")

	evaluator := watcherv1.PodEvaluator{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: watcherv1.PodEvaluatorMetadataName}, &evaluator); err != nil {
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

	pod := corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		l.Error(err, "unable to fetch Pods")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	podNameSlice := strings.Split(pod.Name, "-")
	podBaseName := strings.Join(podNameSlice[:len(podNameSlice)-1], "-")
	if _, ok := podsInChecking[podBaseName]; ok {
		// same type of pod already exists in the map, no need to proceed the following checks
		l.Info("Skip checks for same set of pods", "pod", req.Name, "basename", podBaseName)
		return ctrl.Result{}, nil
	}
	podsInChecking[podBaseName] = &pod

	var resourceRules rules.ResourceRules = evaluator.Spec.Rules
	evaluationResult := resourceRules.EvaluateAll(ctx, req, r.Client, l, pod)
	if evaluationResult.Err != nil {
		l.Error(evaluationResult.Err, "hit error with evaluation")
		return ctrl.Result{}, evaluationResult.Err
	}

	annotations := map[string]string{
		AnnotationCheckedTime: time.Now().Format(time.RFC3339),
		AnnotationIssue:       evaluationResult.IssuesLabelsAsString(),
	}
	pod.SetAnnotations(annotations)
	if err := r.Update(ctx, &pod); err != nil {
		l.Error(err, "unable to update annotations")
	}

	if annotations[AnnotationIssue] != "" {
		msg := evaluationResult.IssueMessagesAsString()
		l.Info(msg)
		if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
			l.Error(err, "Failed to send message to slack", "msg", msg)
		}
	}

	// reset the map
	podsInChecking = map[string]*corev1.Pod{}
	return ctrl.Result{}, nil
}

func (r *PodEvaluatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log
	podsInChecking = map[string]*corev1.Pod{}
	if err := mgr.GetFieldIndexer().IndexField(&corev1.Pod{}, ".metadata.name", func(rawObj runtime.Object) []string {
		pod := rawObj.(*corev1.Pod)
		l.Info("indexing", "pod", pod.Name)
		return []string{pod.Name}
	}); err != nil {
		return err
	}
	l.Info("init manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&watcherv1.PodEvaluator{}).
		For(&corev1.Pod{}).
		WithEventFilter(GetPredicateFuncs(l)).
		Complete(r)
}
