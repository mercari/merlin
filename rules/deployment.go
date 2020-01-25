package rules

import (
	"context"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	DeploymentIssueHasNoEnoughReplica    = "no_enough_replica"
	DeploymentIssueMinReplicaTooLow      = "min_replica_too_low"
	DeploymentIssueHasNoCanaryDeployment = "no_canary_deployment"
)

type DeploymentRules struct {
	HasCanary HasCanary `json:"hasCanary,omitempty"`
	Replica   Replica   `json:"replica,omitempty"`
}

func (r DeploymentRules) EvaluateAll(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, resource interface{}) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	deployment := resource.(appsv1.Deployment)
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
			evaluationResult.Issues = append(evaluationResult.Issues, DeploymentIssueHasNoCanaryDeployment)
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
	if r.Enabled {
		if deployment.Status.AvailableReplicas != *deployment.Spec.Replicas {
			evaluationResult.Issues = append(evaluationResult.Issues, DeploymentIssueHasNoEnoughReplica)
		}
		if *deployment.Spec.Replicas < r.Min {
			evaluationResult.Issues = append(evaluationResult.Issues, DeploymentIssueMinReplicaTooLow)
		}
	}
	return evaluationResult
}
