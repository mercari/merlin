package rules

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type DeploymentRules struct {
	HasCanary HasCanary `json:"hasCanary,omitempty"`
	Replica   Replica   `json:"replica,omitempty"`
}

func (r DeploymentRules) EvaluateAll(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, resource interface{}) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	deployment, ok := resource.(appsv1.Deployment)
	if !ok {
		evaluationResult.Err = fmt.Errorf("unable to convert resource to deployment type")
		return evaluationResult
	}
	evaluationResult.
		Combine(r.HasCanary.Evaluate(ctx, req, cli, log, deployment)).
		Combine(r.Replica.Evaluate(ctx, req, cli, log, deployment))
	return evaluationResult
}

type HasCanary struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (c *HasCanary) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, deployment appsv1.Deployment) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if c.Enabled {
		hasCanaryDeployment := false
		deployments := appsv1.DeploymentList{}
		if err := cli.List(ctx, &deployments, &client.ListOptions{Namespace: req.Namespace}); err != nil {
			evaluationResult.Err = err
			return evaluationResult
		}
		for _, d := range deployments.Items {
			// TODO: better way to check? e.g., generic checks for canary deployment? and what if a service has multiple deployments..
			if strings.HasSuffix(d.Name, "-canary") {
				hasCanaryDeployment = true
			}
		}
		if !hasCanaryDeployment {
			evaluationResult.Issues = append(evaluationResult.Issues, Issue{
				Severity: IssueSeverityInfo,
				Label:    IssueLabelHasNoCanaryDeployment,
				Message:  fmt.Sprintf("Deployment `%s` has no corresponding canary deployment in namespace `%s`", deployment.Name, deployment.Namespace),
			})
		}
	}
	return evaluationResult
}

type Replica struct {
	Min     int32 `json:"min,omitempty"`
	Enabled bool  `json:"enabled,omitempty"`
}

func (r *Replica) Evaluate(ctx context.Context, req ctrl.Request, client client.Client, log logr.Logger, deployment appsv1.Deployment) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if !r.Enabled {
		return evaluationResult
	}
	if deployment.Status.AvailableReplicas != *deployment.Spec.Replicas {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Severity: IssueSeverityWarning,
			Label:    IssueLabelHasNoEnoughReplica,
			Message:  fmt.Sprintf("Deployment `%s` has no enough available replica in namespace `%s`", deployment.Name, deployment.Namespace),
		})
	}
	if *deployment.Spec.Replicas < r.Min {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Severity: IssueSeverityInfo,
			Label:    IssueLabelMinReplicaTooLow,
			Message:  fmt.Sprintf("Deployment `%s` minimal replica is too low (desired: %v, current %v) in namespace `%s`", deployment.Name, r.Min, *deployment.Spec.Replicas, deployment.Namespace),
		})
	}
	return evaluationResult
}
