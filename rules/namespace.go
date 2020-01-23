package rules

import (
	"context"
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

type IstioInjection struct {
	Label string `json:"label,omitempty"`
}

func (r NamespaceRules) Evaluate(ctx context.Context, req ctrl.Request, c client.Client, namespace corev1.Namespace) EvaluationResult {
	var evaluationResult EvaluationResult

	if r.IstioInjection.Label == LabelExists || r.IstioInjection.Label == LabelFalse || r.IstioInjection.Label == LabelTrue {
		istioInjectionLabelExpected := strings.ToLower(r.IstioInjection.Label)
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
