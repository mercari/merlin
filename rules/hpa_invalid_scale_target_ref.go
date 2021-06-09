package rules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mercari/merlin/alert"
	merlinv1beta1 "github.com/mercari/merlin/api/v1beta1"
)

type HPAInvalidScaleTargetRefRule struct {
	rule
	resource *merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef
}

func (h *HPAInvalidScaleTargetRefRule) New(ctx context.Context, cli client.Client, logger logr.Logger, key client.ObjectKey) (Rule, error) {
	h.cli = cli
	h.log = logger
	h.status = &Status{}
	h.resource = &merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{}
	if err := h.cli.Get(ctx, key, h.resource); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *HPAInvalidScaleTargetRefRule) GetObject() runtime.Object {
	return h.resource
}

func (h HPAInvalidScaleTargetRefRule) GetName() string {
	return strings.Join([]string{getStructName(h.resource), h.resource.Name}, Separator)
}

func (h HPAInvalidScaleTargetRefRule) GetObjectMeta() metav1.ObjectMeta {
	return h.resource.ObjectMeta
}

func (h HPAInvalidScaleTargetRefRule) GetNotification() merlinv1beta1.Notification {
	return h.resource.Spec.Notification
}

func (h *HPAInvalidScaleTargetRefRule) SetFinalizer(finalizer string) {
	h.resource.ObjectMeta.Finalizers = append(h.resource.ObjectMeta.Finalizers, finalizer)
}

func (h *HPAInvalidScaleTargetRefRule) RemoveFinalizer(finalizer string) {
	h.resource.ObjectMeta.Finalizers = removeString(h.resource.ObjectMeta.Finalizers, finalizer)
}

func (h *HPAInvalidScaleTargetRefRule) EvaluateAll(ctx context.Context) (alerts []alert.Alert, err error) {
	hpaList := &autoscalingv1.HorizontalPodAutoscalerList{}
	if err = h.cli.List(ctx, hpaList); err != nil {
		return
	}

	if len(hpaList.Items) == 0 {
		h.log.Info("no hpa found")
		return
	}
	for _, hpa := range hpaList.Items {
		h.log.Info("evaluating", fmt.Sprintf("%T", hpa), hpa)
		var a alert.Alert
		a, err = h.Evaluate(ctx, &hpa)
		if err != nil {
			return
		}
		alerts = append(alerts, a)
	}
	return
}

func (h *HPAInvalidScaleTargetRefRule) Evaluate(ctx context.Context, object interface{}) (a alert.Alert, err error) {
	hpa, ok := object.(*autoscalingv1.HorizontalPodAutoscaler)
	if !ok {
		err = fmt.Errorf("object being evaluated is not type %T", hpa)
		return
	}
	h.log.Info("evaluating", fmt.Sprintf("%T", hpa), hpa.Name)
	key := client.ObjectKey{Namespace: hpa.Namespace, Name: hpa.Name}
	a = alert.Alert{
		Suppressed:      h.resource.Spec.Notification.Suppressed,
		Severity:        h.resource.Spec.Notification.Severity,
		MessageTemplate: h.resource.Spec.Notification.CustomMessageTemplate,
		ResourceName:    key.String(),
		ResourceKind:    getStructName(hpa),
		Violated:        false,
	}
	if isStringInSlice(h.resource.Spec.IgnoreNamespaces, key.Namespace) {
		a.Violated = false
		a.Message = "namespace is ignored by the rule"
		return
	}
	var hasMatch bool
	switch hpa.Spec.ScaleTargetRef.Kind {
	case "Deployment":
		deployments := appsv1.DeploymentList{}
		if err = h.cli.List(ctx, &deployments, &client.ListOptions{Namespace: hpa.Namespace}); client.IgnoreNotFound(err) != nil {
			h.log.Error(err, "unable to list", "kind", deployments.Kind)
			return
		}
		for _, d := range deployments.Items {
			if d.Name == hpa.Spec.ScaleTargetRef.Name {
				hasMatch = true
				break
			}
		}
	case "ReplicaSet":
		replicaSets := appsv1.ReplicaSetList{}
		if err = h.cli.List(ctx, &replicaSets, &client.ListOptions{Namespace: hpa.Namespace}); client.IgnoreNotFound(err) != nil {
			h.log.Error(err, "unable to list", "kind", replicaSets.Kind)
			return
		}
		for _, d := range replicaSets.Items {
			if d.Name == hpa.Spec.ScaleTargetRef.Name {
				hasMatch = true
				break
			}
		}
	default:
		err = fmt.Errorf("unknown HPA ScaleTargetRef kind")
		h.log.Error(err, "kind", hpa.Spec.ScaleTargetRef.Kind, "name", hpa.Spec.ScaleTargetRef.Name)
		return
	}

	if hasMatch {
		a.Message = "HPA has valid scale target ref"
	} else {
		a.Violated = true
		a.Message = "HPA has invalid scale target ref"
	}
	h.status.setViolation(key, a.Violated)
	return
}

func (h *HPAInvalidScaleTargetRefRule) GetDelaySeconds(object interface{}) (time.Duration, error) {
	return 0, nil
}
