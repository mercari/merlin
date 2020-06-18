package rules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kouzoh/merlin/alert"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

type secretUnusedRule struct {
	// clusterResource is the api resource this Rule uses
	resource *merlinv1.ClusterRuleSecretUnused
	rule
}

func NewSecretUnusedRule(cli client.Client, logger logr.Logger) Rule {
	return &secretUnusedRule{
		resource: &merlinv1.ClusterRuleSecretUnused{},
		rule: rule{
			cli:    cli,
			log:    logger,
			status: &Status{},
		}}
}

func (s secretUnusedRule) IsInitialized() bool {
	return s.isClusterResourceInitialized
}

func (s *secretUnusedRule) GetObject(ctx context.Context, key client.ObjectKey) (runtime.Object, error) {
	if err := s.cli.Get(ctx, key, s.resource); err != nil {
		if apierrs.IsNotFound(err) {
			s.log.Info("rule does not exist")
			s.isClusterResourceInitialized = false
			return s.resource, nil
		}
		return s.resource, err
	}
	s.isClusterResourceInitialized = true
	return s.resource, nil
}

func (s secretUnusedRule) GetName() string {
	return strings.Join([]string{merlinv1.GetStructName(s.resource), s.resource.Name}, Separator)
}

func (s secretUnusedRule) GetObjectMeta() metav1.ObjectMeta {
	return s.resource.ObjectMeta
}

func (s secretUnusedRule) GetNotification() merlinv1.Notification {
	return s.resource.Spec.Notification
}

func (s *secretUnusedRule) SetFinalizer(finalizer string) {
	s.resource.ObjectMeta.Finalizers = append(s.resource.ObjectMeta.Finalizers, finalizer)
}

func (s *secretUnusedRule) RemoveFinalizer(finalizer string) {
	s.resource.ObjectMeta.Finalizers = removeString(s.resource.ObjectMeta.Finalizers, finalizer)
}

func (s *secretUnusedRule) EvaluateAll(ctx context.Context) (alerts []alert.Alert, err error) {
	secrets := &corev1.SecretList{}
	if err = s.cli.List(ctx, secrets); err != nil {
		return
	}

	if len(secrets.Items) == 0 {
		s.log.Info("no secrets found")
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

func (s *secretUnusedRule) Evaluate(ctx context.Context, object interface{}) (a alert.Alert, err error) {
	secret, isSecret := object.(*corev1.Secret)
	pod, isPod := object.(*corev1.Pod)
	if isSecret && secret.Type == corev1.SecretTypeOpaque {
		return s.evaluateSecret(ctx, secret)
	} else if isPod {
		return s.evaluatePod(ctx, pod)
	}
	err = fmt.Errorf("object being evaluated is not type %T or %T", secret, pod)
	return
}

func (s *secretUnusedRule) evaluatePod(ctx context.Context, pod *corev1.Pod) (a alert.Alert, err error) {
	a = alert.Alert{
		Suppressed:      s.resource.Spec.Notification.Suppressed,
		Severity:        s.resource.Spec.Notification.Severity,
		MessageTemplate: s.resource.Spec.Notification.CustomMessageTemplate,
		Message:         "secret is not being used",
		ResourceKind:    merlinv1.GetStructName(corev1.Secret{}),
		Violated:        true,
	}
	if s.status.CheckedAt == nil || len(s.status.Violations) == 0 {
		a.Violated = false
		return
	}
	for secret := range s.status.Violations {
		key := client.ObjectKey{
			Namespace: strings.Split(secret, Separator)[0],
			Name:      strings.Split(secret, Separator)[1]}
		if pod.Namespace != key.Namespace {
			a.Violated = false
			continue
		}
		a.ResourceName = key.String()
		for _, vol := range pod.Spec.Volumes {
			if vol.Secret != nil && vol.Secret.SecretName == key.Name {
				a.Violated = false
				a.Message = fmt.Sprintf("Secret is being used by pod '%s' volume '%s'", pod.Name, vol.Name)
				s.status.SetViolation(key, a.Violated)
				return
			}
		}
		for _, container := range pod.Spec.Containers {
			for _, envFrom := range container.EnvFrom {
				if envFrom.SecretRef != nil && envFrom.SecretRef.Name == key.Name {
					a.Violated = false
					a.Message = fmt.Sprintf("Secret is being used by pod '%s' container '%s' env", pod.Name, container.Name)
					s.status.SetViolation(key, a.Violated)
					return
				}
			}
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == key.Name {
					a.Violated = false
					a.Message = fmt.Sprintf("Secret is being used by pod '%s' container '%s' env '%s'", pod.Name, container.Name, env.Name)
					s.status.SetViolation(key, a.Violated)
					return
				}
			}
		}
	}
	return
}

func (s *secretUnusedRule) evaluateSecret(ctx context.Context, secret *corev1.Secret) (a alert.Alert, err error) {
	key := client.ObjectKey{Namespace: secret.Namespace, Name: secret.Name}
	a = alert.Alert{
		Suppressed:      s.resource.Spec.Notification.Suppressed,
		Severity:        s.resource.Spec.Notification.Severity,
		MessageTemplate: s.resource.Spec.Notification.CustomMessageTemplate,
		Message:         "secret is not being used",
		ResourceKind:    merlinv1.GetStructName(secret),
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
		s.log.Info("checking pod secrets", "pod", pod.Name)
		for _, vol := range pod.Spec.Volumes {
			if vol.Secret != nil && vol.Secret.SecretName == secret.Name {
				a.Violated = false
				a.Message = fmt.Sprintf("Secret is being used by pod '%s' volume '%s'", pod.Name, vol.Name)
				s.status.SetViolation(key, a.Violated)
				return
			}
		}
		for _, container := range pod.Spec.Containers {
			for _, envFrom := range container.EnvFrom {
				if envFrom.SecretRef != nil && envFrom.SecretRef.Name == secret.Name {
					a.Violated = false
					a.Message = fmt.Sprintf("Secret is being used by pod '%s' container '%s' env", pod.Name, container.Name)
					s.status.SetViolation(key, a.Violated)
					return
				}
			}
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == secret.Name {
					a.Violated = false
					a.Message = fmt.Sprintf("Secret is being used by pod '%s' container '%s' env '%s'", pod.Name, container.Name, env.Name)
					s.status.SetViolation(key, a.Violated)
					return
				}
			}
		}
	}
	s.status.SetViolation(key, a.Violated)
	return

}

func (s *secretUnusedRule) GetDelaySeconds(object interface{}) (time.Duration, error) {
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
