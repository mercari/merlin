package rules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mercari/merlin/alert"
	merlinv1beta1 "github.com/mercari/merlin/api/v1beta1"
)

type ConfigMapUnusedRule struct {
	// resource is the api resource this Rule uses
	resource *merlinv1beta1.ClusterRuleConfigMapUnused
	rule
}

func (s *ConfigMapUnusedRule) New(ctx context.Context, cli client.Client, logger logr.Logger, key client.ObjectKey) (Rule, error) {
	s.cli = cli
	s.log = logger
	s.status = &Status{}
	s.resource = &merlinv1beta1.ClusterRuleConfigMapUnused{}
	if err := s.cli.Get(ctx, key, s.resource); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *ConfigMapUnusedRule) GetObject() runtime.Object {
	return s.resource
}

func (s ConfigMapUnusedRule) GetName() string {
	return strings.Join([]string{getStructName(s.resource), s.resource.Name}, Separator)
}

func (s ConfigMapUnusedRule) GetObjectMeta() metav1.ObjectMeta {
	return s.resource.ObjectMeta
}

func (s ConfigMapUnusedRule) GetNotification() merlinv1beta1.Notification {
	return s.resource.Spec.Notification
}

func (s *ConfigMapUnusedRule) SetFinalizer(finalizer string) {
	s.resource.ObjectMeta.Finalizers = append(s.resource.ObjectMeta.Finalizers, finalizer)
}

func (s *ConfigMapUnusedRule) RemoveFinalizer(finalizer string) {
	s.resource.ObjectMeta.Finalizers = removeString(s.resource.ObjectMeta.Finalizers, finalizer)
}

func (s *ConfigMapUnusedRule) EvaluateAll(ctx context.Context) (alerts []alert.Alert, err error) {
	configMaps := &corev1.ConfigMapList{}
	if err = s.cli.List(ctx, configMaps); err != nil {
		return
	}

	if len(configMaps.Items) == 0 {
		s.log.Info("no resource found")
		return
	}
	for _, configMap := range configMaps.Items {
		s.log.Info("evaluating configMap", "configMap", configMap.Name)
		a, evaluateErr := s.evaluateConfigMap(ctx, &configMap)
		if evaluateErr != nil {
			err = evaluateErr
			return
		}
		alerts = append(alerts, a)
	}
	return
}

func (s *ConfigMapUnusedRule) Evaluate(ctx context.Context, object interface{}) (a alert.Alert, err error) {
	configMap, isConfigMap := object.(*corev1.ConfigMap)
	pod, isPod := object.(*corev1.Pod)
	if isConfigMap {
		return s.evaluateConfigMap(ctx, configMap)
	} else if isPod {
		return s.evaluatePod(ctx, pod)
	}
	err = fmt.Errorf("object being evaluated is not type %T or %T", configMap, pod)
	return
}

func (s *ConfigMapUnusedRule) evaluatePod(ctx context.Context, pod *corev1.Pod) (a alert.Alert, err error) {
	a = alert.Alert{
		Suppressed:      s.resource.Spec.Notification.Suppressed,
		Severity:        s.resource.Spec.Notification.Severity,
		MessageTemplate: s.resource.Spec.Notification.CustomMessageTemplate,
		Message:         "configMap is not being used",
		ResourceKind:    getStructName(corev1.ConfigMap{}),
		Violated:        true,
	}
	if s.status.checkedAt == nil || len(s.status.violations) == 0 {
		a.Violated = false
		return
	}
	for configMap := range s.status.getViolations(pod.Namespace) {
		key := client.ObjectKey{
			Namespace: strings.Split(configMap, Separator)[0],
			Name:      strings.Split(configMap, Separator)[1]}
		a.ResourceName = key.String()
		for _, vol := range pod.Spec.Volumes {
			if vol.ConfigMap != nil && vol.ConfigMap.Name == key.Name {
				a.Violated = false
				a.Message = fmt.Sprintf("ConfigMap is being used by pod '%s' volume '%s'", pod.Name, vol.Name)
				s.status.setViolation(key, a.Violated)
				return
			}
		}
		for _, container := range pod.Spec.Containers {
			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil && envFrom.ConfigMapRef.Name == key.Name {
					a.Violated = false
					a.Message = fmt.Sprintf("ConfigMap is being used by pod '%s' container '%s' env", pod.Name, container.Name)
					s.status.setViolation(key, a.Violated)
					return
				}
			}
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name == key.Name {
					a.Violated = false
					a.Message = fmt.Sprintf("ConfigMap is being used by pod '%s' container '%s' env '%s'", pod.Name, container.Name, env.Name)
					s.status.setViolation(key, a.Violated)
					return
				}
			}
		}
	}
	return
}

func (s *ConfigMapUnusedRule) evaluateConfigMap(ctx context.Context, configMap *corev1.ConfigMap) (a alert.Alert, err error) {
	key := client.ObjectKey{Namespace: configMap.Namespace, Name: configMap.Name}
	a = alert.Alert{
		Suppressed:      s.resource.Spec.Notification.Suppressed,
		Severity:        s.resource.Spec.Notification.Severity,
		MessageTemplate: s.resource.Spec.Notification.CustomMessageTemplate,
		Message:         "configMap is not being used",
		ResourceKind:    getStructName(configMap),
		ResourceName:    key.String(),
		Violated:        true,
	}
	if isStringInSlice(s.resource.Spec.IgnoreNamespaces, configMap.Namespace) {
		a.Violated = false
		a.Message = "namespace is ignored by the rule"
		return
	}

	pods := corev1.PodList{}
	if err = s.cli.List(ctx, &pods, &client.ListOptions{
		Namespace: configMap.Namespace,
	}); err != nil && client.IgnoreNotFound(err) != nil {
		return
	}
	for _, pod := range pods.Items {
		s.log.Info("checking pod configMaps", "pod", pod.Name)
		for _, vol := range pod.Spec.Volumes {
			if vol.ConfigMap != nil && vol.ConfigMap.Name == configMap.Name {
				a.Violated = false
				a.Message = fmt.Sprintf("ConfigMap is being used by pod '%s' volume '%s'", pod.Name, vol.Name)
				s.status.setViolation(key, a.Violated)
				return
			}
		}
		for _, container := range pod.Spec.Containers {
			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil && envFrom.ConfigMapRef.Name == configMap.Name {
					a.Violated = false
					a.Message = fmt.Sprintf("ConfigMap is being used by pod '%s' container '%s' env", pod.Name, container.Name)
					s.status.setViolation(key, a.Violated)
					return
				}
			}
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name == configMap.Name {
					a.Violated = false
					a.Message = fmt.Sprintf("ConfigMap is being used by pod '%s' container '%s' env '%s'", pod.Name, container.Name, env.Name)
					s.status.setViolation(key, a.Violated)
					return
				}
			}
		}
	}
	s.status.setViolation(key, a.Violated)
	return

}

func (s *ConfigMapUnusedRule) GetDelaySeconds(object interface{}) (time.Duration, error) {
	configMap, isConfigMap := object.(*corev1.ConfigMap)
	pod, isPod := object.(*corev1.Pod)
	if isConfigMap {
		delay := configMap.CreationTimestamp.Unix() + s.resource.Spec.InitialDelaySeconds - time.Now().Unix()
		if delay > 0 {
			return time.Duration(delay) * time.Second, nil
		}
		return 0, nil
	} else if isPod {
		return 0, nil
	}
	err := fmt.Errorf("unable to convert object to type %T or %T", configMap, pod)
	return 0, err
}
