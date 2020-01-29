package rules

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodRules struct {
	Restarts            PodRestarts          `json:"restarts,omitempty"`
	OwnedByReplicaset   OwnedByReplicaset    `json:"managedByReplicaset,omitempty"`
	BelongsToService    BelongsToService     `json:"belongsToService,omitempty"`
	ManagedByPDB        ManagedByPDB         `json:"managedByPDB,omitempty"`
	RequiredAnnotations []RequiredAnnotation `json:"requiredAnnotations,omitempty"`
}

// DeepCopyInto is the workaround for similar issue https://github.com/kubernetes/code-generator/issues/52
// the controller-gen cant generate list/map
func (r *PodRules) DeepCopyInto(out *PodRules) {
	*out = *r
	if len(r.RequiredAnnotations) > 0 {
		i, o := &r.RequiredAnnotations, &out.RequiredAnnotations
		*o = make([]RequiredAnnotation, len(*i))
		copy(*o, *i)
	}
	out.Restarts = r.Restarts
	out.OwnedByReplicaset = r.OwnedByReplicaset
	out.BelongsToService = r.BelongsToService
	out.ManagedByPDB = r.ManagedByPDB
}

func (r PodRules) EvaluateAll(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, resource interface{}) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	pod, ok := resource.(corev1.Pod)
	if !ok {
		evaluationResult.Err = fmt.Errorf("unable to convert resource to pod type")
		return evaluationResult
	}
	evaluationResult.
		Combine(r.Restarts.Evaluate(ctx, req, cli, log, pod)).
		Combine(r.OwnedByReplicaset.Evaluate(ctx, req, cli, log, pod)).
		Combine(r.BelongsToService.Evaluate(ctx, req, cli, log, pod)).
		Combine(r.ManagedByPDB.Evaluate(ctx, req, cli, log, pod))

	for _, a := range r.RequiredAnnotations {
		evaluationResult.Combine(a.Evaluate(ctx, req, cli, log, pod))
	}
	return evaluationResult
}

type PodRestarts struct {
	Enabled   bool  `json:"enabled,omitempty"`
	Threshold int32 `json:"threshold,omitempty"`
}

func (r PodRestarts) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, pod corev1.Pod) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if !r.Enabled {
		return evaluationResult
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.RestartCount > r.Threshold && pod.Status.Phase != corev1.PodRunning {
			evaluationResult.Issues = append(evaluationResult.Issues, Issue{
				Severity: IssueSeverityWarning,
				Label:    IssueLabelTooManyRestarts,
				Message:  fmt.Sprintf("Pod `%s` has too many restarts and it's not running", req.NamespacedName),
			})
		}
	}
	return evaluationResult
}

type OwnedByReplicaset struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (r OwnedByReplicaset) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, pod corev1.Pod) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if !r.Enabled || IsPodAJob(pod) {
		return evaluationResult
	}
	// check what replicaset the pod belongs to
	replicaSets := appsv1.ReplicaSetList{}
	ownerReplicaset := ""
	if err := cli.List(ctx, &replicaSets, &client.ListOptions{Namespace: req.Namespace}); err != nil && !apierrs.IsNotFound(err) {
		evaluationResult.Err = fmt.Errorf("unable to fetch replicaSets: %s", err)
	}
	for _, r := range replicaSets.Items {
		matches := 0
		for k, v := range r.Spec.Selector.MatchLabels {
			if _, ok := pod.GetObjectMeta().GetLabels()[k]; ok && v == pod.GetObjectMeta().GetLabels()[k] {
				matches += 1
			}
		}
		if matches == len(r.Spec.Selector.MatchLabels) {
			ownerReplicaset = r.Name
		}
	}

	if ownerReplicaset == "" {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Severity: IssueSeverityWarning,
			Label:    IssueLabelNotOwnedByReplicaset,
			Message:  fmt.Sprintf("Pod `%s` is not managed by a deployment or replicaset", req.NamespacedName),
		})
	}
	return evaluationResult
}

type BelongsToService struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (r BelongsToService) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, pod corev1.Pod) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if !r.Enabled || IsPodAJob(pod) {
		return evaluationResult
	}
	// check what service the pod belongs to
	services := corev1.ServiceList{}
	belongedService := ""
	if err := cli.List(ctx, &services, &client.ListOptions{Namespace: req.Namespace}); err != nil && !apierrs.IsNotFound(err) {
		evaluationResult.Err = fmt.Errorf("unable to fetch services: %s", err)
	}
	for _, s := range services.Items {
		matches := 0
		for k, v := range s.Spec.Selector {
			if _, ok := pod.GetObjectMeta().GetLabels()[k]; ok && v == pod.GetObjectMeta().GetLabels()[k] {
				matches += 1
			}
		}
		if matches == len(s.Spec.Selector) {
			belongedService = s.Name
		}
	}

	if belongedService == "" {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Severity: IssueSeverityWarning,
			Label:    IssueLabelNotBelongToService,
			Message:  fmt.Sprintf("Pod `%s` doesnt belong to any service", req.NamespacedName),
		})
	}
	return evaluationResult
}

type ManagedByPDB struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (r ManagedByPDB) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, pod corev1.Pod) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if !r.Enabled || IsPodAJob(pod) {
		return evaluationResult
	}
	// check what pdb the pod belongs to
	pdbs := policyv1beta1.PodDisruptionBudgetList{}
	managedPDB := ""
	if err := cli.List(ctx, &pdbs, &client.ListOptions{Namespace: req.Namespace}); err != nil && !apierrs.IsNotFound(err) {
		evaluationResult.Err = fmt.Errorf("unable to fetch pdbs: %s", err)
	}
	for _, pdb := range pdbs.Items {
		matches := 0
		for k, v := range pdb.Spec.Selector.MatchLabels {
			if _, ok := pod.GetObjectMeta().GetLabels()[k]; ok && v == pod.GetObjectMeta().GetLabels()[k] {
				matches += 1
			}
		}
		if matches == len(pdb.Spec.Selector.MatchLabels) {
			managedPDB = pdb.Name
		}
	}
	if managedPDB == "" {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Severity: IssueSeverityWarning,
			Label:    IssueLabelNotManagedByPDB,
			Message:  fmt.Sprintf("Pod `%s` is not managed by PDB", req.NamespacedName),
		})
	}
	return evaluationResult
}

type RequiredAnnotation struct {
	Enabled bool   `json:"enabled,omitempty"`
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	// TODO: add regex checks?
}

func (r RequiredAnnotation) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, pod corev1.Pod) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if !r.Enabled || IsPodAJob(pod) {
		return evaluationResult
	}
	podAnnotations := pod.GetAnnotations()
	if v, ok := podAnnotations[r.Key]; !ok {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Severity: IssueSeverityInfo,
			Label:    IssueLabel(fmt.Sprintf(string(IssueLabelMissingAnnotation), r.Key)),
			Message:  fmt.Sprintf("Pod `%s` doesnt have annotation `%s`", req.NamespacedName, r.Key),
		})
	} else if v != r.Value {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Severity: IssueSeverityInfo,
			Label:    IssueLabel(fmt.Sprintf(string(IssueLabelUnexpectedAnnotationValue), r.Key)),
			Message:  fmt.Sprintf("Pod `%s` annotations `%s` value is unexpected", req.NamespacedName, r.Key),
		})
	}

	return evaluationResult
}

// IsPodAJob checks if the pod is a job
func IsPodAJob(pod corev1.Pod) bool {
	isJob := false
	for _, o := range pod.OwnerReferences {
		if o.Kind == "Job" {
			isJob = true
		}
	}
	return isJob
}
