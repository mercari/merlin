package rules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kouzoh/merlin/alert"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

type PDBInvalidSelectorRule struct {
	rule
	resource *merlinv1.ClusterRulePDBInvalidSelector
}

func (s *PDBInvalidSelectorRule) New(ctx context.Context, cli client.Client, logger logr.Logger, key client.ObjectKey) (Rule, error) {
	s.cli = cli
	s.log = logger
	s.status = &Status{}
	s.resource = &merlinv1.ClusterRulePDBInvalidSelector{}
	if err := s.cli.Get(ctx, key, s.resource); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *PDBInvalidSelectorRule) GetObject() runtime.Object {
	return s.resource
}

func (s PDBInvalidSelectorRule) GetName() string {
	return strings.Join([]string{getStructName(s.resource), s.resource.Name}, Separator)
}

func (s PDBInvalidSelectorRule) GetObjectMeta() metav1.ObjectMeta {
	return s.resource.ObjectMeta
}

func (s PDBInvalidSelectorRule) GetNotification() merlinv1.Notification {
	return s.resource.Spec.Notification
}

func (s *PDBInvalidSelectorRule) SetFinalizer(finalizer string) {
	s.resource.ObjectMeta.Finalizers = append(s.resource.ObjectMeta.Finalizers, finalizer)
}

func (s *PDBInvalidSelectorRule) RemoveFinalizer(finalizer string) {
	s.resource.ObjectMeta.Finalizers = removeString(s.resource.ObjectMeta.Finalizers, finalizer)
}

func (s *PDBInvalidSelectorRule) EvaluateAll(ctx context.Context) (alerts []alert.Alert, err error) {
	pdbList := &policyv1beta1.PodDisruptionBudgetList{}
	if err = s.cli.List(ctx, pdbList); err != nil {
		return
	}

	if len(pdbList.Items) == 0 {
		s.log.Info("no hpa found")
		return
	}
	for _, pdb := range pdbList.Items {
		s.log.Info("evaluating", fmt.Sprintf("%T", pdb), pdb)
		var a alert.Alert
		a, err = s.Evaluate(ctx, &pdb)
		if err != nil {
			return
		}
		alerts = append(alerts, a)
	}
	return
}

func (s *PDBInvalidSelectorRule) Evaluate(ctx context.Context, object interface{}) (a alert.Alert, err error) {
	pdb, ok := object.(*policyv1beta1.PodDisruptionBudget)
	if !ok {
		err = fmt.Errorf("object being evaluated is not type %T", pdb)
		return
	}
	s.log.Info("evaluating", fmt.Sprintf("%T", pdb), pdb.Name)
	key := client.ObjectKey{Namespace: pdb.Namespace, Name: pdb.Name}
	a = alert.Alert{
		Suppressed:      s.resource.Spec.Notification.Suppressed,
		Severity:        s.resource.Spec.Notification.Severity,
		MessageTemplate: s.resource.Spec.Notification.CustomMessageTemplate,
		ResourceName:    key.String(),
		ResourceKind:    getStructName(pdb),
		Violated:        false,
	}
	if isStringInSlice(s.resource.Spec.IgnoreNamespaces, key.Namespace) {
		a.Violated = false
		a.Message = "namespace is ignored by the rule"
		return
	}
	pods := corev1.PodList{}
	if err = s.cli.List(ctx, &pods, &client.ListOptions{
		Namespace:     pdb.Namespace,
		LabelSelector: labels.Set(pdb.Spec.Selector.MatchLabels).AsSelector(),
	}); err != nil && client.IgnoreNotFound(err) != nil {
		return
	}
	if len(pods.Items) <= 0 {
		a.Violated = true
		a.Message = "PDB has no matched pods for the selector"
	} else {
		a.Message = "PDB has pods for the selector"
	}
	s.status.setViolation(key, a.Violated)
	return
}

func (s *PDBInvalidSelectorRule) GetDelaySeconds(object interface{}) (time.Duration, error) {
	return 0, nil
}
