package v1

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// +kubebuilder:object:generate=false

type Rule interface {
	Evaluate(ctx context.Context, cli client.Client, log logr.Logger, resource interface{}) *EvaluationResult
}

// RequiredLabel is the
type RequiredLabel struct {
	// Key is the label key name
	Key string `json:"key"`
	// Value is the label value, when match is set as "regexp", the acceptable syntax of regex is RE2 (https://github.com/google/re2/wiki/Syntax)
	Value string `json:"value"`
	// Match is the way of matching, default to "exact" match, can also use "regexp" and set value to a regular express for matching.
	Match string `json:"match,omitempty"`
}

func (r RequiredLabel) Validate(labels map[string]string) (issue Issue, err error) {
	v, ok := labels[r.Key]
	if !ok {
		issue.Label = IssueLabelNoRequiredLabel
		issue.DefaultMessage = fmt.Sprintf("Namespace doenst have required label %s", r.Key)
		return
	}
	if r.Match == "" || r.Match == "exact" {
		if v != r.Value {
			issue.Label = IssueLabelIncorrectRequiredLabelValue
			issue.DefaultMessage = fmt.Sprintf("Namespace has incorrect label value %s (expect %s) for label %s", v, r.Value, r.Key)
			return
		}
	} else if r.Match == "regexp" {
		var re *regexp.Regexp
		re, err = regexp.Compile(r.Value)
		if err != nil {
			return
		}
		if len(re.FindAllString(v, -1)) <= 0 {
			issue.Label = IssueLabelIncorrectRequiredLabelValue
			issue.DefaultMessage = fmt.Sprintf("Namespace has incorrect label value %s (regex match %s) for label %s", v, r.Value, r.Key)
			return
		}
	}
	return
}

type Selector struct {
	// Name is the resource name this selector will select
	Name string `json:"name,omitempty"`
	// MatchLabels is the map of labels this selector will select on
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

func (s *Selector) IsLabelMatched(resourceLabels map[string]string) bool {
	for k, v := range s.MatchLabels {
		if rv, ok := resourceLabels[k]; rv != v || !ok {
			return false
		}
	}
	return true
}

type Notification struct {
	// Notifiers is the list of notifiers for this notification to send
	Notifiers []string `json:"notifiers"`
	// Suppressed means if this notification has been suppressed, useful for temporary
	Suppressed bool `json:"suppressed,omitempty"`
	// Severity is the severity of the issue, one of info, warning, critical, or fatal
	Severity IssueSeverity `json:"severity,omitempty"`
	// CustomMessageTemplate can used for customized message, variables can be used are "ResourceName, Severity, and DefaultMessage"
	CustomMessageTemplate string `json:"customMessageTemplate,omitempty"`
}

// +kubebuilder:object:generate=false
// IssueSeverity indicates the severity of the issue
type IssueSeverity string

const (
	IssueSeverityDefault  IssueSeverity = ""
	IssueSeverityFatal    IssueSeverity = "fatal"
	IssueSeverityCritical IssueSeverity = "critical"
	IssueSeverityWarning  IssueSeverity = "warning"
	IssueSeverityInfo     IssueSeverity = "info"
)

// +kubebuilder:object:generate=false
// IssueLabel is the label for the issue, in shorter text.
type IssueLabel string

const (
	IssueLabelNone                        IssueLabel = ""
	IssueLabelHasNoEnoughReplica          IssueLabel = "no_enough_replica"
	IssueLabelMinReplicaTooLow            IssueLabel = "min_replica_too_low"
	IssueLabelHasNoCanaryDeployment       IssueLabel = "no_canary_deployment"
	IssueLabelHighReplicaPercent          IssueLabel = "high_replica_percent"
	IssueLabelInvalidSetting              IssueLabel = "invalid_setting"
	IssueLabelInvalidScaleTargetRef       IssueLabel = "invalid_scale_target_ref"
	IssueLabelNoRequiredLabel             IssueLabel = "no_required_label"
	IssueLabelIncorrectRequiredLabelValue IssueLabel = "incorrect_required_label_value"
	IssueLabelNoMatchedPods               IssueLabel = "no_matched_pods"
	IssueLabelTooManyRestarts             IssueLabel = "too_many_restarts"
	IssueLabelNotOwnedByReplicaset        IssueLabel = "not_owned_by_replicaset"
	IssueLabelNotBelongToService          IssueLabel = "not_belonged_to_service"
	IssueLabelNotManagedByPDB             IssueLabel = "not_managed_by_pdb"
	IssueLabelMissingAnnotation           IssueLabel = "missing_annotation_%s"
	IssueLabelUnexpectedAnnotationValue   IssueLabel = "unexpected_annotation_value_%s"
)

// +kubebuilder:object:generate=false
// Issue is the problem found by the rules
type Issue struct {
	Label          IssueLabel
	DefaultMessage string
	Notification   Notification
}

// String returns the string of the issue, combined with severity and message.
func (i Issue) String() string {
	return fmt.Sprintf("[%s] %s", i.Notification.Severity, i.DefaultMessage)
}

// +kubebuilder:object:generate=false
// EvaluationResult is the result after evaluation
type EvaluationResult struct {
	// NamespacedName is the resource name with namespace prefixed.
	NamespacedName types.NamespacedName
	// Err is the operational error (code or api problems, not the resource issue.)
	Err error
	// Issues are the list of issues from the evaluation
	Issues []Issue
}

// String returns joined labels for all issues in the results
func (e *EvaluationResult) String() string {
	issues := make([]string, len(e.Issues))
	for i, issue := range e.Issues {
		issues[i] = string(issue.Label)
	}
	return strings.Join(issues, ";")
}

// Combine takes results and combine them
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
