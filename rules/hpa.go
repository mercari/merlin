package rules

import (
	"context"
	"fmt"
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
	hpa, ok := resource.(autoscalingv1.HorizontalPodAutoscaler)
	if !ok {
		evaluationResult.Err = fmt.Errorf("unable to convert resource to hpa type")
		return evaluationResult
	}
	evaluationResult.
		Combine(r.ReachedMaxReplica.Evaluate(ctx, req, cli, log, hpa)).
		Combine(r.InvalidMetrics.Evaluate(ctx, req, cli, log, hpa))
	return evaluationResult
}

type ReachedMaxReplica struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (r ReachedMaxReplica) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, hpa autoscalingv1.HorizontalPodAutoscaler) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if !r.Enabled {
		return evaluationResult
	}
	if hpa.Spec.MaxReplicas == hpa.Status.CurrentReplicas {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Severity: IssueSeverityWarning,
			Label:    IssueLabelReachedMaxReplica,
			Message:  fmt.Sprintf("HPA `%s` reached its max replicas in namespace `%s`", hpa.Name, hpa.Namespace),
		})
	}
	return evaluationResult
}

type InvalidMetrics struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (r InvalidMetrics) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, hpa autoscalingv1.HorizontalPodAutoscaler) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if !r.Enabled {
		return evaluationResult
	}
	if hpa.Status.CurrentCPUUtilizationPercentage == nil {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Severity: IssueSeverityCritical,
			Label:    IssueLabelInvalidSetting,
			Message:  fmt.Sprintf("HPA `%s` config is not setup properly in namespace `%s`", hpa.Name, hpa.Namespace),
		})
	}
	return evaluationResult
}
