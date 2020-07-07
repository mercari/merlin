package rules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kouzoh/merlin/alert"
	merlinv1beta1 "github.com/kouzoh/merlin/api/v1beta1"
)

type HPAReplicaPercentageRule struct{}

func (h *HPAReplicaPercentageRule) New(ctx context.Context, cli client.Client, logger logr.Logger, key client.ObjectKey) (Rule, error) {
	var r Rule
	if key.Namespace == "" {
		resource := &merlinv1beta1.ClusterRuleHPAReplicaPercentage{}
		if err := cli.Get(ctx, key, resource); err != nil {
			return nil, err
		}
		r = &hpaReplicaPercentageClusterRule{
			resource: resource,
			rule:     rule{cli: cli, log: logger, status: &Status{}},
		}
	} else {
		resource := &merlinv1beta1.RuleHPAReplicaPercentage{}
		if err := cli.Get(ctx, key, resource); err != nil {
			return nil, err
		}
		r = &hpaReplicaPercentageNamespaceRule{
			resource: resource,
			rule:     rule{cli: cli, log: logger, status: &Status{}},
		}
	}
	return r, nil
}

type hpaReplicaPercentageClusterRule struct {
	rule
	resource *merlinv1beta1.ClusterRuleHPAReplicaPercentage
}

func (h *hpaReplicaPercentageClusterRule) GetObject() runtime.Object {
	return h.resource
}

func (h hpaReplicaPercentageClusterRule) GetName() string {
	return strings.Join([]string{getStructName(h.resource), h.resource.Name}, Separator)
}

func (h hpaReplicaPercentageClusterRule) GetObjectMeta() metav1.ObjectMeta {
	return h.resource.ObjectMeta
}

func (h hpaReplicaPercentageClusterRule) GetNotification() merlinv1beta1.Notification {
	return h.resource.Spec.Notification
}

func (h *hpaReplicaPercentageClusterRule) SetFinalizer(finalizer string) {
	h.resource.ObjectMeta.Finalizers = append(h.resource.ObjectMeta.Finalizers, finalizer)
}

func (h *hpaReplicaPercentageClusterRule) RemoveFinalizer(finalizer string) {
	h.resource.ObjectMeta.Finalizers = removeString(h.resource.ObjectMeta.Finalizers, finalizer)
}

func (h *hpaReplicaPercentageClusterRule) EvaluateAll(ctx context.Context) (alerts []alert.Alert, err error) {
	hpaList := &autoscalingv1.HorizontalPodAutoscalerList{}
	if err = h.cli.List(ctx, hpaList); err != nil {
		return
	}

	if len(hpaList.Items) == 0 {
		h.log.Info("no hpa found")
		return
	}
	for _, hpa := range hpaList.Items {
		h.log.V(1).Info("evaluating", fmt.Sprintf("%T", hpa), hpa)
		a, e := h.Evaluate(ctx, &hpa)
		if e != nil {
			err = e
			return
		}
		alerts = append(alerts, a)
	}
	return
}

func (h *hpaReplicaPercentageClusterRule) Evaluate(ctx context.Context, object interface{}) (a alert.Alert, err error) {
	hpa, ok := object.(*autoscalingv1.HorizontalPodAutoscaler)
	if !ok {
		err = fmt.Errorf("object being evaluated is not type %T", hpa)
		return
	}
	h.log.V(1).Info("evaluating", fmt.Sprintf("%T", hpa), hpa.Name)
	key := client.ObjectKey{Namespace: hpa.Namespace, Name: hpa.Name}
	a = alert.Alert{
		Suppressed:      h.resource.Spec.Notification.Suppressed,
		Severity:        h.resource.Spec.Notification.Severity,
		MessageTemplate: h.resource.Spec.Notification.CustomMessageTemplate,
		Message:         fmt.Sprintf("HPA percentage is within threshold (< %v%%)", h.resource.Spec.Percent),
		ResourceName:    key.String(),
		ResourceKind:    getStructName(hpa),
		Violated:        false,
	}
	if isStringInSlice(h.resource.Spec.IgnoreNamespaces, key.Namespace) {
		a.Violated = false
		a.Message = "namespace is ignored by the rule"
		return
	}
	if float64(hpa.Status.CurrentReplicas)/float64(hpa.Spec.MaxReplicas) >= float64(h.resource.Spec.Percent)/100.0 {
		a.Violated = true
		a.Message = fmt.Sprintf("HPA percentage is >= %v%%", h.resource.Spec.Percent)
	}
	h.status.setViolation(key, a.Violated)
	return
}

func (h *hpaReplicaPercentageClusterRule) GetDelaySeconds(object interface{}) (time.Duration, error) {
	return 0, nil
}

type hpaReplicaPercentageNamespaceRule struct {
	rule
	resource *merlinv1beta1.RuleHPAReplicaPercentage
}

func (h *hpaReplicaPercentageNamespaceRule) GetObject() runtime.Object {
	return h.resource
}

func (h hpaReplicaPercentageNamespaceRule) GetName() string {
	return strings.Join([]string{getStructName(h.resource), h.resource.Name}, Separator)
}

func (h hpaReplicaPercentageNamespaceRule) GetObjectMeta() metav1.ObjectMeta {
	return h.resource.ObjectMeta
}

func (h hpaReplicaPercentageNamespaceRule) GetNotification() merlinv1beta1.Notification {
	return h.resource.Spec.Notification
}

func (h *hpaReplicaPercentageNamespaceRule) SetFinalizer(finalizer string) {
	h.resource.ObjectMeta.Finalizers = append(h.resource.ObjectMeta.Finalizers, finalizer)
}

func (h *hpaReplicaPercentageNamespaceRule) RemoveFinalizer(finalizer string) {
	h.resource.ObjectMeta.Finalizers = removeString(h.resource.ObjectMeta.Finalizers, finalizer)
}

func (h *hpaReplicaPercentageNamespaceRule) EvaluateAll(ctx context.Context) (alerts []alert.Alert, err error) {
	hpaList := &autoscalingv1.HorizontalPodAutoscalerList{}
	if err = h.cli.List(ctx, hpaList, getListOptions(h.resource.Spec.Selector, h.resource.Namespace)); err != nil {
		return
	}

	if len(hpaList.Items) == 0 {
		h.log.Info("no hpa found")
		return
	}
	for _, hpa := range hpaList.Items {
		h.log.V(1).Info("evaluating", fmt.Sprintf("%T", hpa), hpa)
		a, e := h.Evaluate(ctx, &hpa)
		if e != nil {
			err = e
			return
		}
		alerts = append(alerts, a)
	}
	return
}

func (h *hpaReplicaPercentageNamespaceRule) Evaluate(ctx context.Context, object interface{}) (a alert.Alert, err error) {
	hpa, ok := object.(*autoscalingv1.HorizontalPodAutoscaler)
	if !ok {
		err = fmt.Errorf("object being evaluated is not type %T", hpa)
		return
	}
	h.log.V(1).Info("evaluating", fmt.Sprintf("%T", hpa), hpa.Name)
	key := client.ObjectKey{Namespace: hpa.Namespace, Name: hpa.Name}
	a = alert.Alert{
		Suppressed:      h.resource.Spec.Notification.Suppressed,
		Severity:        h.resource.Spec.Notification.Severity,
		MessageTemplate: h.resource.Spec.Notification.CustomMessageTemplate,
		Message:         fmt.Sprintf("HPA percentage is within threshold (< %v%%)", h.resource.Spec.Percent),
		ResourceName:    key.String(),
		ResourceKind:    getStructName(hpa),
		Violated:        false,
	}
	if float64(hpa.Status.CurrentReplicas)/float64(hpa.Spec.MaxReplicas) >= float64(h.resource.Spec.Percent)/100.0 {
		a.Violated = true
		a.Message = fmt.Sprintf("HPA percentage is >= %v%%", h.resource.Spec.Percent)
	}
	h.status.setViolation(key, a.Violated)
	return
}

func (h *hpaReplicaPercentageNamespaceRule) GetDelaySeconds(object interface{}) (time.Duration, error) {
	return 0, nil
}
