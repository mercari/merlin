package rules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kouzoh/merlin/alert"
	merlinv1beta1 "github.com/kouzoh/merlin/api/v1beta1"
)

type ServiceInvalidSelectorRule struct {
	rule
	resource *merlinv1beta1.ClusterRuleServiceInvalidSelector
}

func (s *ServiceInvalidSelectorRule) New(ctx context.Context, cli client.Client, logger logr.Logger, key client.ObjectKey) (Rule, error) {
	s.cli = cli
	s.log = logger
	s.status = &Status{}
	s.resource = &merlinv1beta1.ClusterRuleServiceInvalidSelector{}
	if err := s.cli.Get(ctx, key, s.resource); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *ServiceInvalidSelectorRule) GetObject() runtime.Object {
	return s.resource
}

func (s ServiceInvalidSelectorRule) GetName() string {
	return strings.Join([]string{getStructName(s.resource), s.resource.Name}, Separator)
}

func (s ServiceInvalidSelectorRule) GetObjectMeta() metav1.ObjectMeta {
	return s.resource.ObjectMeta
}

func (s ServiceInvalidSelectorRule) GetNotification() merlinv1beta1.Notification {
	return s.resource.Spec.Notification
}

func (s *ServiceInvalidSelectorRule) SetFinalizer(finalizer string) {
	s.resource.ObjectMeta.Finalizers = append(s.resource.ObjectMeta.Finalizers, finalizer)
}

func (s *ServiceInvalidSelectorRule) RemoveFinalizer(finalizer string) {
	s.resource.ObjectMeta.Finalizers = removeString(s.resource.ObjectMeta.Finalizers, finalizer)
}

func (s *ServiceInvalidSelectorRule) EvaluateAll(ctx context.Context) (alerts []alert.Alert, err error) {
	serviceList := &corev1.ServiceList{}
	if err = s.cli.List(ctx, serviceList); err != nil {
		return
	}

	if len(serviceList.Items) == 0 {
		s.log.Info("no services found")
		return
	}
	for _, svc := range serviceList.Items {
		s.log.Info("evaluating", fmt.Sprintf("%T", svc), svc)
		var a alert.Alert
		a, err = s.Evaluate(ctx, &svc)
		if err != nil {
			return
		}
		alerts = append(alerts, a)
	}
	return
}

func (s *ServiceInvalidSelectorRule) Evaluate(ctx context.Context, object interface{}) (a alert.Alert, err error) {
	svc, ok := object.(*corev1.Service)
	if !ok {
		err = fmt.Errorf("object being evaluated is not type %T", svc)
		return
	}
	s.log.Info("evaluating", fmt.Sprintf("%T", svc), svc.Name)
	key := client.ObjectKey{Namespace: svc.Namespace, Name: svc.Name}
	a = alert.Alert{
		Suppressed:      s.resource.Spec.Notification.Suppressed,
		Severity:        s.resource.Spec.Notification.Severity,
		MessageTemplate: s.resource.Spec.Notification.CustomMessageTemplate,
		ResourceName:    key.String(),
		ResourceKind:    getStructName(svc),
		Violated:        false,
	}
	if isStringInSlice(s.resource.Spec.IgnoreNamespaces, key.Namespace) {
		a.Violated = false
		a.Message = "namespace is ignored by the rule"
		return
	}
	pods := corev1.PodList{}
	if err = s.cli.List(ctx, &pods, &client.ListOptions{
		Namespace:     svc.Namespace,
		LabelSelector: labels.Set(svc.Spec.Selector).AsSelector(),
	}); err != nil && client.IgnoreNotFound(err) != nil {
		return
	}
	if len(pods.Items) <= 0 {
		a.Violated = true
		a.Message = "Service has no matched pods for the selector"
	} else {
		a.Message = "Service has pods for the selector"
	}
	s.status.setViolation(key, a.Violated)
	return
}

func (s *ServiceInvalidSelectorRule) GetDelaySeconds(object interface{}) (time.Duration, error) {
	return 0, nil
}
