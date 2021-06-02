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

	"github.com/kouzoh/merlin/alert"
	merlinv1beta1 "github.com/kouzoh/merlin/api/v1beta1"
)

type SecretUnusedRule struct {
	// resource is the api resource this Rule uses
	resource *merlinv1beta1.ClusterRuleSecretUnused
	rule
}

func (s *SecretUnusedRule) New(ctx context.Context, cli client.Client, logger logr.Logger, key client.ObjectKey) (Rule, error) {
	s.cli = cli
	s.log = logger
	s.status = &Status{}
	s.resource = &merlinv1beta1.ClusterRuleSecretUnused{}
	if err := s.cli.Get(ctx, key, s.resource); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SecretUnusedRule) GetObject() runtime.Object {
	return s.resource
}

func (s SecretUnusedRule) GetName() string {
	return strings.Join([]string{getStructName(s.resource), s.resource.Name}, Separator)
}

func (s SecretUnusedRule) GetObjectMeta() metav1.ObjectMeta {
	return s.resource.ObjectMeta
}

func (s SecretUnusedRule) GetNotification() merlinv1beta1.Notification {
	return s.resource.Spec.Notification
}

func (s *SecretUnusedRule) SetFinalizer(finalizer string) {
	s.resource.ObjectMeta.Finalizers = append(s.resource.ObjectMeta.Finalizers, finalizer)
}

func (s *SecretUnusedRule) RemoveFinalizer(finalizer string) {
	s.resource.ObjectMeta.Finalizers = removeString(s.resource.ObjectMeta.Finalizers, finalizer)
}

func (s *SecretUnusedRule) EvaluateAll(ctx context.Context) (alerts []alert.Alert, err error) {
	secrets := &corev1.SecretList{}
	if err = s.cli.List(ctx, secrets); err != nil {
		return
	}

	if len(secrets.Items) == 0 {
		s.log.Info("no resource found")
		return
	}
	for _, secret := range secrets.Items {
		if secret.Type != corev1.SecretTypeOpaque {
			continue
		}
		s.log.Info("evaluating secret", "secret", secret.Name)
		a, evaluateErr := s.evaluateSecret(ctx, &secret)
		if evaluateErr != nil {
			err = evaluateErr
			return
		}
		alerts = append(alerts, a)
	}
	return
}

func (s *SecretUnusedRule) Evaluate(ctx context.Context, object interface{}) (a alert.Alert, err error) {
	secret, isSecret := object.(*corev1.Secret)
	pod, isPod := object.(*corev1.Pod)
	if isSecret {
		if secret.Type != corev1.SecretTypeOpaque {
			return
		}
		return s.evaluateSecret(ctx, secret)
	} else if isPod {
		return s.evaluatePod(ctx, pod)
	}
	err = fmt.Errorf("object being evaluated is not type %T or %T", secret, pod)
	return
}

func (s *SecretUnusedRule) evaluatePod(ctx context.Context, pod *corev1.Pod) (a alert.Alert, err error) {
	a = alert.Alert{
		Suppressed:      s.resource.Spec.Notification.Suppressed,
		Severity:        s.resource.Spec.Notification.Severity,
		MessageTemplate: s.resource.Spec.Notification.CustomMessageTemplate,
		Message:         "secret is not being used",
		ResourceKind:    getStructName(corev1.Secret{}),
		Violated:        true,
	}
	if s.status.checkedAt == nil || len(s.status.violations) == 0 {
		a.Violated = false
		return
	}
	for secret := range s.status.getViolations(pod.Namespace) {
		key := client.ObjectKey{
			Namespace: strings.Split(secret, Separator)[0],
			Name:      strings.Split(secret, Separator)[1]}
		a.ResourceName = key.String()
		for _, vol := range pod.Spec.Volumes {
			if vol.Secret != nil && vol.Secret.SecretName == key.Name {
				a.Violated = false
				a.Message = fmt.Sprintf("Secret is being used by pod '%s' volume '%s'", pod.Name, vol.Name)
				s.status.setViolation(key, a.Violated)
				return
			}
		}
		for _, container := range pod.Spec.Containers {
			for _, envFrom := range container.EnvFrom {
				if envFrom.SecretRef != nil && envFrom.SecretRef.Name == key.Name {
					a.Violated = false
					a.Message = fmt.Sprintf("Secret is being used by pod '%s' container '%s' env", pod.Name, container.Name)
					s.status.setViolation(key, a.Violated)
					return
				}
			}
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == key.Name {
					a.Violated = false
					a.Message = fmt.Sprintf("Secret is being used by pod '%s' container '%s' env '%s'", pod.Name, container.Name, env.Name)
					s.status.setViolation(key, a.Violated)
					return
				}
			}
		}
	}
	return
}

func (s *SecretUnusedRule) evaluateSecret(ctx context.Context, secret *corev1.Secret) (a alert.Alert, err error) {
	key := client.ObjectKey{Namespace: secret.Namespace, Name: secret.Name}
	a = alert.Alert{
		Suppressed:      s.resource.Spec.Notification.Suppressed,
		Severity:        s.resource.Spec.Notification.Severity,
		MessageTemplate: s.resource.Spec.Notification.CustomMessageTemplate,
		Message:         "secret is not being used",
		ResourceKind:    getStructName(secret),
		ResourceName:    key.String(),
		Violated:        true,
	}
	if isStringInSlice(s.resource.Spec.IgnoreNamespaces, secret.Namespace) {
		a.Violated = false
		a.Message = "namespace is ignored by the rule"
		return
	}

	pods := corev1.PodList{}
	if err = s.cli.List(ctx, &pods, &client.ListOptions{
		Namespace: secret.Namespace,
	}); err != nil && client.IgnoreNotFound(err) != nil {
		return
	}
	for _, pod := range pods.Items {
		s.log.V(1).Info("checking pod secrets", "pod", pod.Name)
		for _, vol := range pod.Spec.Volumes {
			if vol.Secret != nil && vol.Secret.SecretName == secret.Name {
				a.Violated = false
				a.Message = fmt.Sprintf("Secret is being used by pod '%s' volume '%s'", pod.Name, vol.Name)
				s.status.setViolation(key, a.Violated)
				return
			}
		}
		for _, container := range pod.Spec.Containers {
			for _, envFrom := range container.EnvFrom {
				if envFrom.SecretRef != nil && envFrom.SecretRef.Name == secret.Name {
					a.Violated = false
					a.Message = fmt.Sprintf("Secret is being used by pod '%s' container '%s' env", pod.Name, container.Name)
					s.status.setViolation(key, a.Violated)
					return
				}
			}
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == secret.Name {
					a.Violated = false
					a.Message = fmt.Sprintf("Secret is being used by pod '%s' container '%s' env '%s'", pod.Name, container.Name, env.Name)
					s.status.setViolation(key, a.Violated)
					return
				}
			}
		}
	}
	s.status.setViolation(key, a.Violated)
	return

}

func (s *SecretUnusedRule) GetDelaySeconds(object interface{}) (time.Duration, error) {
	secret, isSecret := object.(*corev1.Secret)
	pod, isPod := object.(*corev1.Pod)
	if isSecret {
		delay := secret.CreationTimestamp.Unix() + s.resource.Spec.InitialDelaySeconds - time.Now().Unix()
		if delay > 0 {
			return time.Duration(delay) * time.Second, nil
		}
		return 0, nil
	} else if isPod {
		return 0, nil
	}
	err := fmt.Errorf("unable to convert object to type %T or %T", secret, pod)
	return 0, err
}
