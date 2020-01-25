package rules

import (
	"context"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	LabelExists                                      = "exists"
	LabelTrue                                        = "true"
	LabelFalse                                       = "false"
	IstioInjectionLabelKey                           = "istio-injection"
	NamespaceIssueNoIstioInjectionLabel              = "no_istio_injection_label"
	NamespaceIssueUnexpectedIstioInjectionLabelValue = "unexpected_istio_injection_label_value"
)

type NamespaceRules struct {
	IstioInjection IstioInjection `json:"istioInjection,omitempty"`
}

func (r NamespaceRules) EvaluateAll(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, resource interface{}) *EvaluationResult {
	namespace := resource.(corev1.Namespace)
	evaluationResult := r.IstioInjection.Evaluate(ctx, req, cli, log, namespace)
	return evaluationResult
}

type IstioInjection struct {
	Label string `json:"label,omitempty"`
}

func (r IstioInjection) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, namespace corev1.Namespace) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if r.Label == LabelExists || r.Label == LabelFalse || r.Label == LabelTrue {
		istioInjectionLabelExpected := strings.ToLower(r.Label)
		istioInjectionLabel, ok := namespace.Labels[IstioInjectionLabelKey]
		if !ok {
			evaluationResult.Issues = append(evaluationResult.Issues, NamespaceIssueNoIstioInjectionLabel)
		}

		if (istioInjectionLabelExpected == LabelTrue || istioInjectionLabelExpected == LabelFalse) &&
			istioInjectionLabel != istioInjectionLabelExpected {
			evaluationResult.Issues = append(evaluationResult.Issues, NamespaceIssueUnexpectedIstioInjectionLabelValue)
		}
	}

	return evaluationResult
}
