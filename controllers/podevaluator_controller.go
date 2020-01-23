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
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	watcherv1 "github.com/kouzoh/merlin/api/v1"
)

// PodEvaluatorReconciler reconciles a PodEvaluator object
type PodEvaluatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// TODO: better naming..?
type PodInfo struct {
	Name       string
	NameSpace  string
	Deployment string
	ReplicaSet string
	Service    string
	HPA        string
	PDB        string
}

// TODO: better way of handling this - it's used to coordinate b/w reconciliation processes.
var podInfos = map[string]*PodInfo{}

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

	pods := corev1.PodList{}
	if err := r.List(ctx, &pods, &client.ListOptions{Namespace: req.Namespace}); err != nil && !apierrs.IsNotFound(err) {
		l.Error(err, "unable to fetch Pods")
		return ctrl.Result{}, err
	}

	for _, p := range pods.Items {
		// check if pod has too many restarts and not running
		for _, containerStatus := range p.Status.ContainerStatuses {
			if containerStatus.RestartCount > evaluator.Spec.Restarts && p.Status.Phase != corev1.PodRunning {
				msg := fmt.Sprintf("Pod `%s` has too many restarts and it's not running", req.NamespacedName)
				l.Info(msg, "pod", req.Name, "restart limit", evaluator.Spec.Restarts)
				if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
					l.Error(err, "Failed to send message to slack")
				}
			}
		}

		// below are checks only needed for one pod from the same sets of pods
		podNameSlice := strings.Split(p.Name, "-")
		podBaseName := strings.Join(podNameSlice[:len(podNameSlice)-1], "-")
		if _, ok := podInfos[podBaseName]; ok {
			// same type of pod already exists in the map, no need to proceed the following checks
			l.Info("Skip some checks for same set of pods", "pod", req.Name, "basename", podBaseName)
			continue
		}
		info := PodInfo{Name: podBaseName}
		podInfos[podBaseName] = &info

		// check what deployment the pod belongs to
		deployments := appsv1.DeploymentList{}
		if err := r.List(ctx, &deployments, &client.ListOptions{Namespace: req.Namespace}); err != nil && !apierrs.IsNotFound(err) {
			l.Error(err, "unable to fetch Deployments")
			return ctrl.Result{}, err
		}
		for _, d := range deployments.Items {
			matches := 0
			for k, v := range d.Spec.Selector.MatchLabels {
				if _, ok := p.GetObjectMeta().GetLabels()[k]; ok && v == p.GetObjectMeta().GetLabels()[k] {
					matches += 1
				}
			}
			if matches == len(d.Spec.Selector.MatchLabels) {
				info.Deployment = d.Name
			}
		}

		// check what replicaset the pod belongs to
		replicaSets := appsv1.ReplicaSetList{}
		if err := r.List(ctx, &replicaSets, &client.ListOptions{Namespace: req.Namespace}); err != nil && !apierrs.IsNotFound(err) {
			l.Error(err, "unable to fetch replicaSets")
			return ctrl.Result{}, err
		}
		for _, r := range replicaSets.Items {
			matches := 0
			for k, v := range r.Spec.Selector.MatchLabels {
				if _, ok := p.GetObjectMeta().GetLabels()[k]; ok && v == p.GetObjectMeta().GetLabels()[k] {
					matches += 1
				}
			}
			if matches == len(r.Spec.Selector.MatchLabels) {
				info.ReplicaSet = r.Name
			}
		}

		if info.Deployment == "" && info.ReplicaSet == "" {
			msg := fmt.Sprintf("Pod `%s` is not managed by a deployment or replicaset", req.NamespacedName)
			l.Info(msg, "pod", req.Name)
			if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
				l.Error(err, "Failed to send message to slack")
			}
		}

		// check what service the pod belongs to
		services := corev1.ServiceList{}
		if err := r.List(ctx, &services, &client.ListOptions{Namespace: req.Namespace}); err != nil && !apierrs.IsNotFound(err) {
			l.Error(err, "unable to fetch services")
			return ctrl.Result{}, err
		}
		for _, s := range services.Items {
			matches := 0
			for k, v := range s.Spec.Selector {
				if _, ok := p.GetObjectMeta().GetLabels()[k]; ok && v == p.GetObjectMeta().GetLabels()[k] {
					matches += 1
				}
			}
			if matches == len(s.Spec.Selector) {
				info.Service = s.Name
			}
		}

		if info.Service == "" {
			isJob := false
			for _, o := range p.OwnerReferences {
				if o.Kind == "Job" {
					isJob = true
				}
			}
			if !isJob {
				msg := fmt.Sprintf("Pod `%s` is not used by a service", req.NamespacedName)
				l.Info(msg, "pod", req.Name)
				if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
					l.Error(err, "Failed to send message to slack")
				}
			}
		}

		// check what pdb the pod belongs to
		pdbs := policyv1beta1.PodDisruptionBudgetList{}
		if err := r.List(ctx, &pdbs, &client.ListOptions{Namespace: req.Namespace}); err != nil && !apierrs.IsNotFound(err) {
			l.Error(err, "unable to fetch pdbs")
			return ctrl.Result{}, err
		}
		for _, pdb := range pdbs.Items {
			matches := 0
			for k, v := range pdb.Spec.Selector.MatchLabels {
				if _, ok := p.GetObjectMeta().GetLabels()[k]; ok && v == p.GetObjectMeta().GetLabels()[k] {
					matches += 1
				}
			}
			l.Info("pdb", "pdb", pdb.Name)
			if matches == len(pdb.Spec.Selector.MatchLabels) {
				info.PDB = pdb.Name
			}
		}
		if info.PDB == "" {
			msg := fmt.Sprintf("Pod `%s` is not managed by PDB", req.NamespacedName)
			l.Info(msg, "pod", req.Name)
			if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
				l.Error(err, "Failed to send message to slack")
			}
		}

		// check if the pod's replicaset or deployment has hpa
		hpas := autoscalingv1.HorizontalPodAutoscalerList{}
		if err := r.List(ctx, &hpas, &client.ListOptions{Namespace: req.Namespace}); err != nil && !apierrs.IsNotFound(err) {
			l.Error(err, "unable to fetch hpas")
			return ctrl.Result{}, err
		}
		for _, hpa := range hpas.Items {
			if hpa.Spec.ScaleTargetRef.Kind == "Deployment" {
				if info.Deployment == hpa.Spec.ScaleTargetRef.Name {
					info.HPA = hpa.Name
				}
			} else if hpa.Spec.ScaleTargetRef.Kind == "ReplicaSet" {
				if info.ReplicaSet == hpa.Spec.ScaleTargetRef.Name {
					info.HPA = hpa.Name
				}
			}
		}
		if info.HPA == "" {
			msg := fmt.Sprintf("Pod `%s` is not managed by HPA", req.NamespacedName)
			l.Info(msg, "pod", req.Name)
			if err := notifiers.Spec.Slack.SendMessage(msg); err != nil {
				l.Error(err, "Failed to send message to slack")
			}
		}

	}

	return ctrl.Result{}, nil
}

func (r *PodEvaluatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	l := r.Log
	podInfos = map[string]*PodInfo{}
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
