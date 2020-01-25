package rules

import (
	"context"
	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HPARules struct {
	ReachedMaxReplica ReachedMaxReplica `json:"reachedMaxReplica,omitempty"`
	InvalidMetrics    InvalidMetrics    `json:"invalidMetrics,omitempty"`
}

func (r HPARules) EvaluateAll(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, resource interface{}) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	hpa := resource.(autoscalingv1.HorizontalPodAutoscaler)
	evaluationResult.
		Combine(r.ReachedMaxReplica.Evaluate(ctx, req, cli, log, hpa).
			Combine(r.InvalidMetrics.Evaluate(ctx, req, cli, log, hpa)))
	return evaluationResult
}

type ReachedMaxReplica struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (r ReachedMaxReplica) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, hpa autoscalingv1.HorizontalPodAutoscaler) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if r.Enabled {
		if hpa.Spec.MaxReplicas == hpa.Status.CurrentReplicas {
			evaluationResult.Issues = append(evaluationResult.Issues, "HPA reached its max replicas")
		}
	}
	return evaluationResult
}

type InvalidMetrics struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (r InvalidMetrics) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, hpa autoscalingv1.HorizontalPodAutoscaler) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if r.Enabled {
		if hpa.Status.CurrentCPUUtilizationPercentage == nil {
			evaluationResult.Issues = append(evaluationResult.Issues, "HPA config is not setup properly")
		}
	}
	return evaluationResult
}
