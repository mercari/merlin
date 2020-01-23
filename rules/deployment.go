package rules

import (
	"context"
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
	Canary  Canary  `json:"canary,omitempty"`
	Replica Replica `json:"replica,omitempty"`
}

type Canary struct {
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled,omitempty"`
}

type Replica struct {
	Description string `json:"description,omitempty"`
	Min         int32  `json:"min,omitempty"`
	Enabled     bool   `json:"enabled,omitempty"`
}

func (r DeploymentRules) Evaluate(ctx context.Context, req ctrl.Request, c client.Client, deployment appsv1.Deployment) EvaluationResult {
	var evaluationResult EvaluationResult

	if r.Replica.Enabled {
		if deployment.Status.AvailableReplicas != *deployment.Spec.Replicas {
			evaluationResult.Issues = append(evaluationResult.Issues, DeploymentIssueHasNoEnoughReplica)
		}
		if *deployment.Spec.Replicas < r.Replica.Min {
			evaluationResult.Issues = append(evaluationResult.Issues, DeploymentIssueMinReplicaTooLow)
		}
	}

	if r.Canary.Enabled {
		hasCanaryDeployment := false
		deployments := appsv1.DeploymentList{}
		if err := c.List(ctx, &deployments, &client.ListOptions{Namespace: req.Namespace}); err != nil {
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
