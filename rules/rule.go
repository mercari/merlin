package rules

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kouzoh/merlin/alert"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

const Separator = string(types.Separator)

type Status struct {
	// CheckedAt is the latest time this status was updated
	CheckedAt *time.Time
	// Violations is the resources violated the rule, with object key as names.
	Violations map[string]time.Time
}

func (r *Status) SetViolation(key client.ObjectKey, isViolated bool) {
	now := time.Now()
	if r.Violations == nil {
		r.Violations = map[string]time.Time{}
	}
	delete(r.Violations, key.String())
	if isViolated {
		r.Violations[key.String()] = time.Now()
	}
	r.CheckedAt = &now
}

type Rule interface {
	// IsInitialized returns if the rule has be initialized, if not means the user has not created the rules yet
	IsInitialized() bool
	// IsReady returns if the rule is ready to be used, RuleReconciler should initialize the rule and run evaluations for the first time
	IsReady() bool
	// SetReady sets the rule's ready status
	SetReady(bool)
	// GetName returns the rule's name
	GetName() string
	// GetObject returns the runtime.Object of the rule
	GetObject(ctx context.Context, key client.ObjectKey) (runtime.Object, error)
	// GetObjectMeta returns the GetObjectMeta of the rule
	GetObjectMeta() metav1.ObjectMeta
	// GetNotification returns the notifications specified for the rule
	GetNotification() merlinv1.Notification
	// EvaluateAll evaluates all applicable resources for the rule, it'll be called by RuleReconciler
	EvaluateAll(ctx context.Context) ([]alert.Alert, error)
	// Evaluate evaluates single resource, it'll be called by ResourceReconciler
	Evaluate(ctx context.Context, watchedResource interface{}) (alert.Alert, error)
	// SetFinalizer sets finalizer for the rule
	SetFinalizer(finalizer string)
	// RemoveFinalizer removes the finalizer from the rule
	RemoveFinalizer(finalizer string)
	//GetDelaySeconds returns the delayed time before the rule should be evaluated
	GetDelaySeconds(object interface{}) (time.Duration, error)
}

type rule struct {
	cli client.Client
	log logr.Logger
	// isReady is the value	 indicates if this rule is ready
	isReady bool
	// status is the status of this rule
	status *Status
	//
	isClusterResourceInitialized bool
	//
	isNamespaceResourceInitialized bool
}

func (r *rule) IsReady() bool {
	return r.isReady
}

func (r *rule) SetReady(isReady bool) {
	r.isReady = isReady
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

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

// isStringInSlice checks if a string is in a slice
func isStringInSlice(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
