package rules

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	LabelExists            = "exists"
	LabelTrue              = "true"
	LabelFalse             = "false"
	IstioInjectionLabelKey = "istio-injection"
)

type NamespaceRules struct {
	IstioInjection IstioInjection `json:"istioInjection,omitempty"`
}

func (r NamespaceRules) EvaluateAll(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, resource interface{}) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	namespace, ok := resource.(corev1.Namespace)
	if !ok {
		evaluationResult.Err = fmt.Errorf("unable to convert resource to namespace type")
		return evaluationResult
	}
	evaluationResult.Combine(r.IstioInjection.Evaluate(ctx, req, cli, log, namespace))
	return evaluationResult
}

type IstioInjection struct {
	Label   string `json:"label,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
}

func (r IstioInjection) Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, namespace corev1.Namespace) *EvaluationResult {
	evaluationResult := &EvaluationResult{}
	if !r.Enabled {
		return evaluationResult
	}

	if r.Label == LabelExists || r.Label == LabelFalse || r.Label == LabelTrue {
		istioInjectionLabelExpected := strings.ToLower(r.Label)
		istioInjectionLabel, ok := namespace.Labels[IstioInjectionLabelKey]
		if !ok {
			evaluationResult.Issues = append(evaluationResult.Issues, Issue{
				Severity: IssueSeverityInfo,
				Label:    IssueLabelNoIstioInjectionLabel,
			})
		}

		if (istioInjectionLabelExpected == LabelTrue || istioInjectionLabelExpected == LabelFalse) &&
			istioInjectionLabel != istioInjectionLabelExpected {
			evaluationResult.Issues = append(evaluationResult.Issues, Issue{
				Severity: IssueSeverityInfo,
				Label:    IssueLabelUnexpectedIstioInjectionLabelValue,
			})
		}
	}
	return evaluationResult
}
