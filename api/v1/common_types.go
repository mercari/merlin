package v1

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/kouzoh/merlin/notifiers/alert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// +kubebuilder:object:generate=false

type Rule interface {
	Evaluate(ctx context.Context, cli client.Client, log logr.Logger, resource types.NamespacedName, notifiers map[string]*Notifier) error
	GetName() string
	GetStatus() RuleStatus
	GetGeneration() int64
	GetObjectKind() schema.ObjectKind
	DeepCopyObject() runtime.Object
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

func (r RequiredLabel) Validate(labels map[string]string) (violation string, err error) {
	v, ok := labels[r.Key]
	if !ok {
		return fmt.Sprintf("doenst have required label `%s`", r.Key), nil
	}
	if r.Match == "" || r.Match == "exact" {
		if v != r.Value {
			return fmt.Sprintf("has incorrect label value `%s` (expect `%s`) for label `%s`", v, r.Value, r.Key), nil
		}
	} else if r.Match == "regexp" {
		var re *regexp.Regexp
		re, err = regexp.Compile(r.Value)
		if err != nil {
			return
		}
		if len(re.FindAllString(v, -1)) <= 0 {
			return fmt.Sprintf("has incorrect label value `%s` (regex match `%s`) for label `%s`", v, r.Value, r.Key), nil
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
	// Suppressed means if this notification has been suppressed, used for temporary reduced the noise
	Suppressed bool `json:"suppressed,omitempty"`
	// Severity is the severity of the issue, one of info, warning, critical, or fatal
	Severity alert.Severity `json:"severity,omitempty"`
	// CustomMessageTemplate can used for customized message, variables can be used are "ResourceName, Severity, and ViolationMessage"
	CustomMessageTemplate string `json:"customMessageTemplate,omitempty"`
}

type RuleStatus struct {
	CheckedAt  string            `json:"checkedAt,omitempty"`
	Violations map[string]string `json:"violations,omitempty"`
}

func (r *RuleStatus) SetViolation(namespacedName types.NamespacedName, isViolated bool) {
	if r.Violations == nil {
		r.Violations = map[string]string{}
	}
	delete(r.Violations, namespacedName.String())
	if isViolated {
		r.Violations[namespacedName.String()] = time.Now().Format(time.RFC3339)
	}
	r.CheckedAt = time.Now().Format(time.RFC3339)
}
