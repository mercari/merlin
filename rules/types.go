package rules

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type ResourceRules interface {
	EvaluateAll(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, resource interface{}) *EvaluationResult
}

// IssueSeverity indicates the severity of the issue
type IssueSeverity string

const (
	IssueSeverityFatal    IssueSeverity = "fatal"
	IssueSeverityCritical IssueSeverity = "critical"
	IssueSeverityWarning  IssueSeverity = "warning"
	IssueSeverityInfo     IssueSeverity = "info"
)

// IssueLabel is the label for the issue, in shorter text.
type IssueLabel string

const (
	IssueLabelHasNoEnoughReplica                 IssueLabel = "no_enough_replica"
	IssueLabelMinReplicaTooLow                   IssueLabel = "min_replica_too_low"
	IssueLabelHasNoCanaryDeployment              IssueLabel = "no_canary_deployment"
	IssueLabelReachedMaxReplica                  IssueLabel = "reached_max_replica"
	IssueLabelInvalidSetting                     IssueLabel = "invalid_setting"
	IssueLabelNoIstioInjectionLabel              IssueLabel = "no_istio_injection_label"
	IssueLabelUnexpectedIstioInjectionLabelValue IssueLabel = "unexpected_istio_injection_label_value"
	IssueLabelNoMatchedPods                      IssueLabel = "no_matched_pods"
	IssueLabelTooManyRestarts                    IssueLabel = "too_many_restarts"
	IssueLabelNotOwnedByReplicaset               IssueLabel = "not_owned_by_replicaset"
	IssueLabelNotBelongToService                 IssueLabel = "not_belonged_to_service"
	IssueLabelNotManagedByPDB                    IssueLabel = "not_managed_by_pdb"
)

// Issue is the problem found by the rules
type Issue struct {
	// TODO: customizable severity
	Severity IssueSeverity
	Label    IssueLabel
	Message  string
}

// String returns the string of the issue, combined with severity and message.
func (i Issue) String() string {
	return fmt.Sprintf("[%s] %s", i.Severity, i.Message)
}

// EvaluationResult is the result after evaluation
type EvaluationResult struct {
	// Err is the operational error (code or api problems, not the resource issue.)
	Err error
	// Issues are the list of issues from the evaluation
	Issues []Issue
}

func (e *EvaluationResult) IssueMessagesAsString() string {
	messages := make([]string, len(e.Issues))
	for _, issue := range e.Issues {
		messages = append(messages, issue.Message)
	}
	return strings.Join(messages, ";")
}

func (e *EvaluationResult) IssuesLabelsAsString() string {
	issues := make([]string, len(e.Issues))
	for _, issue := range e.Issues {
		issues = append(issues, issue.String())
	}
	return strings.Join(issues, ";")
}

func (e *EvaluationResult) Combine(a *EvaluationResult) *EvaluationResult {
	var err error
	if e.Err != nil {
		err = e.Err
	}
	if a.Err != nil {
		if err != nil {
			// both are not nil, combine them
			e.Err = fmt.Errorf("%s, %s", e.Err.Error(), a.Err.Error())
		} else {
			err = a.Err
		}
	}
	e.Err = err

	for _, i := range a.Issues {
		e.Issues = append(e.Issues, i)
	}
	return e
}
