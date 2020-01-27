package rules

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PDBRules struct {
	NoMatchedPods PDBNoMatchedPods `json:"noMatchedPods,omitempty"`
}

func (r PDBRules) EvaluateAll(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, resource interface{}) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	pdb, ok := resource.(policyv1beta1.PodDisruptionBudget)
	if !ok {
		evaluationResult.Err = fmt.Errorf("unable to convert resource to PDB type")
		return evaluationResult
	}
	evaluationResult.Combine(r.NoMatchedPods.Evaluate(ctx, req, cli, log, pdb))
	return evaluationResult
}

type PDBNoMatchedPods struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (r PDBNoMatchedPods) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, pdb policyv1beta1.PodDisruptionBudget) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if !r.Enabled {
		return evaluationResult
	}
	pdbSelector := v1.SetAsLabelSelector(pdb.Spec.Selector.MatchLabels).String()
	pods := corev1.PodList{}
	if err := cli.List(ctx, &pods, &client.ListOptions{
		Namespace: req.Namespace,
		Raw: &v1.ListOptions{
			LabelSelector: pdbSelector,
		},
	}); err != nil && !apierrs.IsNotFound(err) {
		evaluationResult.Err = fmt.Errorf("unable to fetch pods: %s", err)
		return evaluationResult
	}
	if len(pods.Items) == 0 {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Severity: IssueSeverityWarning,
			Label:    IssueLabelNoMatchedPods,
			Message:  fmt.Sprintf("PDB `%s` has no target pods in namespace `%s`", pdb.Name, pdb.Namespace),
		})
	}
	return evaluationResult
}
