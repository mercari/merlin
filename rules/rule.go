package rules

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mercari/merlin/alert"
	merlinv1beta1 "github.com/mercari/merlin/api/v1beta1"
)

const Separator = string(types.Separator)

type Status struct {
	sync.Mutex
	// checkedAt is the latest time this status was updated
	checkedAt *time.Time
	// violations is the resources violated the rule, with object key as names.
	violations map[string]time.Time
}

func (r *Status) setViolation(key client.ObjectKey, isViolated bool) {
	now := time.Now()
	if r.violations == nil {
		r.violations = map[string]time.Time{}
	}
	r.Lock()
	delete(r.violations, key.String())
	if isViolated {
		r.violations[key.String()] = time.Now()
	}
	r.Unlock()
	r.checkedAt = &now
}

func (r *Status) getViolations(namespace string) map[string]time.Time {
	violations := map[string]time.Time{}
	r.Lock()
	for k, v := range r.violations {
		if strings.Split(k, Separator)[0] == namespace {
			violations[k] = v
		}
	}
	r.Unlock()
	return violations
}

// RuleFactory is the factory that creates rule
type RuleFactory interface {
	New(context.Context, client.Client, logr.Logger, client.ObjectKey) (Rule, error)
}

// Rule is the interface for reconciler to evaluate if the resource meets the rule's requirement
type Rule interface {
	// IsReady returns if the rule is ready to be used, RuleReconciler should initialize the rule and run evaluations for the first time
	IsReady() bool
	// SetReady sets the rule's ready status
	SetReady(bool)
	// GetName returns the rule's name
	GetName() string
	// GetObject returns the runtime.Object of the rule
	GetObject() runtime.Object
	// GetObjectMeta returns the GetObjectMeta of the rule
	GetObjectMeta() metav1.ObjectMeta
	// GetNotification returns the notifications specified for the rule
	GetNotification() merlinv1beta1.Notification
	// EvaluateAll evaluates all applicable resources for the rule, it'll be called by RuleReconciler
	EvaluateAll(context.Context) ([]alert.Alert, error)
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
}

// IsReady returns if the rule is ready to evaluate
func (r *rule) IsReady() bool {
	return r.isReady
}

// SetReady sets the readiness of the rule.
func (r *rule) SetReady(isReady bool) {
	r.isReady = isReady
}

// removeString removes a string from a slice of string
func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

// isStringInSlice checks if a string is in a slice of string
func isStringInSlice(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// getStrutName returns the name of the struct, handles pointer struct too.
func getStructName(v interface{}) string {
	if t := reflect.TypeOf(v); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}

func validateRequiredLabel(r merlinv1beta1.RequiredLabel, labels map[string]string) (message string, err error) {
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
func getListOptions(s merlinv1beta1.Selector, namespace string) (opts *client.ListOptions) {
	opts = &client.ListOptions{Namespace: namespace}
	if s.Name != "" {
		opts.FieldSelector = fields.Set{".metadata.name": s.Name}.AsSelector()
	}
	if len(s.MatchLabels) != 0 {
		opts.LabelSelector = labels.SelectorFromSet(s.MatchLabels)
	}
	return
}
