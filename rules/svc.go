package rules

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceRules struct {
	NoMatchedPods ServiceNoMatchedPods `json:"noMatchedPods,omitempty"`
}

func (r ServiceRules) EvaluateAll(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, resource interface{}) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	svc, ok := resource.(corev1.Service)
	if !ok {
		evaluationResult.Err = fmt.Errorf("unable to convert resource to Service type")
		return evaluationResult
	}
	evaluationResult.Combine(r.NoMatchedPods.Evaluate(ctx, req, cli, log, svc))
	return evaluationResult
}

type ServiceNoMatchedPods struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (r ServiceNoMatchedPods) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, svc corev1.Service) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if !r.Enabled {
		return evaluationResult
	}
	selector, err := v1.LabelSelectorAsSelector(v1.SetAsLabelSelector(labels.Set(svc.Spec.Selector)))
	if err != nil {
		evaluationResult.Err = fmt.Errorf("unable to convert labelSelctor as selector")
		return evaluationResult
	}
	pods := corev1.PodList{}
	if err := cli.List(ctx, &pods, &client.ListOptions{
		Namespace:     req.Namespace,
		LabelSelector: selector,
	}); err != nil && !apierrs.IsNotFound(err) {
		evaluationResult.Err = fmt.Errorf("unable to fetch pods: %s", err)
		return evaluationResult
	}
	if len(pods.Items) == 0 {
		evaluationResult.Issues = append(evaluationResult.Issues, Issue{
			Severity: IssueSeverityCritical,
			Label:    IssueLabelNoMatchedPods,
			Message:  fmt.Sprintf("Service `%s` has no target pods in namespace `%s`", svc.Name, svc.Namespace),
		})
	}
	return evaluationResult
}
