package v1

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/kouzoh/merlin/alert"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:object:generate=false

// Rule is the interfaces for rules
type Rule interface {
	// Evaluate takes the object and evaluate kubernetes objects based on on the rule. It has client so can retrieve other objects when needed.
	Evaluate(ctx context.Context, cli client.Client, log logr.Logger, object interface{}) (isViolated bool, message string, err error)
	// GetNotification returns Notification that user specified for the rule.
	GetNotification() Notification
	// GetName returns the name of the rule
	GetName() string
	// GetStatus returns the status of the rule
	GetStatus() RuleStatus
	// SetViolationStatus sets the status with violated objects
	SetViolationStatus(name types.NamespacedName, isViolated bool)
	// List returns the List, similar to other SpecList, it can be used to retrieve list of objects from kubernetes API
	List() RuleList
	// IsNamespacedRule returns if the rule is namespaced
	IsNamespacedRule() bool
	// IsNamespaceIgnored returns if the namespace is ignored,
	// note for namespaces its Namespace value is empty, and Name value is the namespace
	// namespaced rule should always return false
	IsNamespaceIgnored(namespace string) bool
	// GetNamespacedRuleList returns RuleList of of its namespacedRule, this is used to determine if a cluster rule has
	// also an associated namespaced rule, if so the reconcile checks if there's any namespaced rule exists before checking
	// cluster rule, since currently if there's any associated namespaced rule exists in a namespace, cluster rule will be ignored.
	// only used by cluster rule, namespaced rule should always return nil
	GetNamespacedRuleList() RuleList
	// GetResourceList returns the ResourceList that the rule cares about, please also take a look of list types.
	// e.g., v1 service rule cares only corev1 services, so the ResourceList is coreV1ServiceList
	GetResourceList() ResourceList
	// GetSelector returns Selector, only used by namespaced rule, cluster rule should return nil
	GetSelector() *Selector
	// GetObjectKind returns schema.ObjectKind, required for calling kuberentes api and creating the instance
	GetObjectKind() schema.ObjectKind
	// DeepCopyObject returns runtime.Object, required for calling kuberentes api and creating the instance
	DeepCopyObject() runtime.Object
	// GetObjectNamespacedName returns the object as types.NamespacedName
	GetObjectNamespacedName(object interface{}) (types.NamespacedName, error)
	// GetObjectMeta returns the objectMeta of the rule
	GetObjectMeta() metav1.ObjectMeta
	// SetFinalizer sets the finalizer in objectMeta
	SetFinalizer(finalizer string)
	// RemoveFinalizer removes the finalizer from objectMeta
	RemoveFinalizer(finalizer string)
}

// +kubebuilder:object:generate=false

type RuleList interface {
	// ListItems returns list of rules
	ListItems() []Rule
	// GetObjectKind returns schema.ObjectKind, required for calling kuberentes api and creating the instance
	GetObjectKind() schema.ObjectKind
	// DeepCopyObject returns runtime.Object, required for calling kuberentes api and creating the instance
	DeepCopyObject() runtime.Object
}

// +kubebuilder:object:generate=false

type ResourceList interface {
	ListItems() []interface{}
	List() runtime.Object
	AddItem(namespacedName types.NamespacedName)
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

// Selector is the resource selector that used when listing kubernetes objects, only namespaced rules have this since cluster rules apply for all objects.
type Selector struct {
	// Name is the resource name this selector will select
	Name string `json:"name,omitempty"`
	// MatchLabels is the map of labels this selector will select on
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// AsListOption returns the selector as client.ListOptions that can be directly used for calling kuberentes api and retrieve objects
func (s *Selector) AsListOption(namespace string) (opts *client.ListOptions) {
	opts = &client.ListOptions{Namespace: namespace}
	if s.Name != "" {
		opts.FieldSelector = fields.Set{".metadata.name": s.Name}.AsSelector()
	}
	if len(s.MatchLabels) != 0 {
		opts.LabelSelector = labels.SelectorFromSet(s.MatchLabels)
	}
	return
}

type Notification struct {
	// Notifiers is the list of notifiers for this notification to send
	Notifiers []string `json:"notifiers"`
	// Suppressed means if this notification has been suppressed, used for temporary reduced the noise
	Suppressed bool `json:"suppressed,omitempty"`
	// Severity is the severity of the issue, one of info, warning, critical, or fatal
	Severity alert.Severity `json:"severity,omitempty"`
	// CustomMessageTemplate can used for customized message, variables can be used are "ResourceName, Severity, and Message"
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
