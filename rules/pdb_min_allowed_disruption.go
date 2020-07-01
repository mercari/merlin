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
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kouzoh/merlin/alert"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

type PDBMinAllowedDisruptionRule struct{}

func (p *PDBMinAllowedDisruptionRule) New(ctx context.Context, cli client.Client, logger logr.Logger, key client.ObjectKey) (Rule, error) {
	var r Rule
	if key.Namespace == "" {
		resource := &merlinv1.ClusterRulePDBMinAllowedDisruption{}
		if err := cli.Get(ctx, key, resource); err != nil {
			return nil, err
		}
		r = &pdbMinAllowedDisruptionClusterRule{
			resource: resource,
			rule:     rule{cli: cli, log: logger, status: &Status{}},
		}
	} else {
		resource := &merlinv1.RulePDBMinAllowedDisruption{}
		if err := cli.Get(ctx, key, resource); err != nil {
			return nil, err
		}
		r = &pdbMinAllowedDisruptionNamespaceRule{
			resource: resource,
			rule:     rule{cli: cli, log: logger, status: &Status{}},
		}
	}
	return r, nil
}

type pdbMinAllowedDisruptionClusterRule struct {
	rule
	resource *merlinv1.ClusterRulePDBMinAllowedDisruption
}

func (p *pdbMinAllowedDisruptionClusterRule) GetObject() runtime.Object {
	return p.resource
}

func (p pdbMinAllowedDisruptionClusterRule) GetName() string {
	return strings.Join([]string{getStructName(p.resource), p.resource.Name}, Separator)
}

func (p pdbMinAllowedDisruptionClusterRule) GetObjectMeta() metav1.ObjectMeta {
	return p.resource.ObjectMeta
}

func (p pdbMinAllowedDisruptionClusterRule) GetNotification() merlinv1.Notification {
	return p.resource.Spec.Notification
}

func (p *pdbMinAllowedDisruptionClusterRule) SetFinalizer(finalizer string) {
	p.resource.ObjectMeta.Finalizers = append(p.resource.ObjectMeta.Finalizers, finalizer)
}

func (p *pdbMinAllowedDisruptionClusterRule) RemoveFinalizer(finalizer string) {
	p.resource.ObjectMeta.Finalizers = removeString(p.resource.ObjectMeta.Finalizers, finalizer)
}

func (p *pdbMinAllowedDisruptionClusterRule) EvaluateAll(ctx context.Context) (alerts []alert.Alert, err error) {
	pdbList := &policyv1beta1.PodDisruptionBudgetList{}
	if err = p.cli.List(ctx, pdbList); err != nil {
		return
	}

	if len(pdbList.Items) == 0 {
		p.log.Info("no pdb found")
		return
	}
	for _, pdb := range pdbList.Items {
		p.log.V(1).Info("evaluating", fmt.Sprintf("%T", pdb), pdb)
		a, e := p.Evaluate(ctx, &pdb)
		if e != nil {
			err = e
			return
		}
		alerts = append(alerts, a)
	}
	return
}

func (p *pdbMinAllowedDisruptionClusterRule) Evaluate(ctx context.Context, object interface{}) (a alert.Alert, err error) {
	pdb, ok := object.(*policyv1beta1.PodDisruptionBudget)
	if !ok {
		err = fmt.Errorf("object being evaluated is not type %T", pdb)
		return
	}
	p.log.V(1).Info("evaluating", fmt.Sprintf("%T", pdb), pdb.Name)
	key := client.ObjectKey{Namespace: pdb.Namespace, Name: pdb.Name}
	a = alert.Alert{
		Suppressed:      p.resource.Spec.Notification.Suppressed,
		Severity:        p.resource.Spec.Notification.Severity,
		MessageTemplate: p.resource.Spec.Notification.CustomMessageTemplate,
		Message:         "",
		ResourceName:    key.String(),
		ResourceKind:    getStructName(pdb),
		Violated:        false,
	}
	if isStringInSlice(p.resource.Spec.IgnoreNamespaces, key.Namespace) {
		a.Violated = false
		a.Message = "namespace is ignored by the rule"
		return
	}
	minAllowedDisruption := 1 // default value
	if p.resource.Spec.MinAllowedDisruption > minAllowedDisruption {
		minAllowedDisruption = p.resource.Spec.MinAllowedDisruption
	}

	var allowedDisruption int
	pods := corev1.PodList{}
	if err = p.cli.List(ctx, &pods, &client.ListOptions{
		Namespace:     pdb.Namespace,
		LabelSelector: labels.SelectorFromSet(pdb.Spec.Selector.MatchLabels),
	}); err != nil && client.IgnoreNotFound(err) != nil {
		return
	}
	if pdb.Spec.MaxUnavailable != nil {
		if allowedDisruption, err = intstr.GetValueFromIntOrPercent(pdb.Spec.MaxUnavailable, len(pods.Items), true); err != nil {
			return
		}
	} else if pdb.Spec.MinAvailable != nil {
		var minAvailable int
		if minAvailable, err = intstr.GetValueFromIntOrPercent(pdb.Spec.MinAvailable, len(pods.Items), true); err != nil {
			return
		}
		allowedDisruption = len(pods.Items) - minAvailable
	}

	if allowedDisruption < minAllowedDisruption {
		a.Violated = true
		a.Message = fmt.Sprintf("PDB doesnt have enough disruption pod (expect %v, but currently is %v)", minAllowedDisruption, allowedDisruption)
	} else {
		a.Message = fmt.Sprintf("PDB has enough disruption pod (expect %v, currently is %v)", minAllowedDisruption, allowedDisruption)
	}
	return
}

func (p *pdbMinAllowedDisruptionClusterRule) GetDelaySeconds(object interface{}) (time.Duration, error) {
	return 0, nil
}

type pdbMinAllowedDisruptionNamespaceRule struct {
	rule
	resource *merlinv1.RulePDBMinAllowedDisruption
}

func (p *pdbMinAllowedDisruptionNamespaceRule) GetObject() runtime.Object {
	return p.resource
}

func (p pdbMinAllowedDisruptionNamespaceRule) GetName() string {
	return strings.Join([]string{getStructName(p.resource), p.resource.Name}, Separator)
}

func (p pdbMinAllowedDisruptionNamespaceRule) GetObjectMeta() metav1.ObjectMeta {
	return p.resource.ObjectMeta
}

func (p pdbMinAllowedDisruptionNamespaceRule) GetNotification() merlinv1.Notification {
	return p.resource.Spec.Notification
}

func (p *pdbMinAllowedDisruptionNamespaceRule) SetFinalizer(finalizer string) {
	p.resource.ObjectMeta.Finalizers = append(p.resource.ObjectMeta.Finalizers, finalizer)
}

func (p *pdbMinAllowedDisruptionNamespaceRule) RemoveFinalizer(finalizer string) {
	p.resource.ObjectMeta.Finalizers = removeString(p.resource.ObjectMeta.Finalizers, finalizer)
}

func (p *pdbMinAllowedDisruptionNamespaceRule) EvaluateAll(ctx context.Context) (alerts []alert.Alert, err error) {
	pdbList := &policyv1beta1.PodDisruptionBudgetList{}
	if err = p.cli.List(ctx, pdbList, getListOptions(p.resource.Spec.Selector, p.resource.Namespace)); err != nil {
		return
	}

	if len(pdbList.Items) == 0 {
		p.log.Info("no pdb found")
		return
	}
	for _, pdb := range pdbList.Items {
		p.log.V(1).Info("evaluating", fmt.Sprintf("%T", pdb), pdb)
		a, e := p.Evaluate(ctx, &pdb)
		if e != nil {
			err = e
			return
		}
		alerts = append(alerts, a)
	}
	return
}

func (p *pdbMinAllowedDisruptionNamespaceRule) Evaluate(ctx context.Context, object interface{}) (a alert.Alert, err error) {
	pdb, ok := object.(*policyv1beta1.PodDisruptionBudget)
	if !ok {
		err = fmt.Errorf("object being evaluated is not type %T", pdb)
		return
	}
	p.log.V(1).Info("evaluating", fmt.Sprintf("%T", pdb), pdb.Name)
	key := client.ObjectKey{Namespace: pdb.Namespace, Name: pdb.Name}
	a = alert.Alert{
		Suppressed:      p.resource.Spec.Notification.Suppressed,
		Severity:        p.resource.Spec.Notification.Severity,
		MessageTemplate: p.resource.Spec.Notification.CustomMessageTemplate,
		Message:         "",
		ResourceName:    key.String(),
		ResourceKind:    getStructName(pdb),
		Violated:        false,
	}
	minAllowedDisruption := 1 // default value
	if p.resource.Spec.MinAllowedDisruption > minAllowedDisruption {
		minAllowedDisruption = p.resource.Spec.MinAllowedDisruption
	}

	var allowedDisruption int
	pods := corev1.PodList{}
	if err = p.cli.List(ctx, &pods, &client.ListOptions{
		Namespace:     pdb.Namespace,
		LabelSelector: labels.SelectorFromSet(pdb.Spec.Selector.MatchLabels),
	}); err != nil && client.IgnoreNotFound(err) != nil {
		return
	}
	if pdb.Spec.MaxUnavailable != nil {
		if allowedDisruption, err = intstr.GetValueFromIntOrPercent(pdb.Spec.MaxUnavailable, len(pods.Items), true); err != nil {
			return
		}
	} else if pdb.Spec.MinAvailable != nil {
		var minAvailable int
		if minAvailable, err = intstr.GetValueFromIntOrPercent(pdb.Spec.MinAvailable, len(pods.Items), true); err != nil {
			return
		}
		allowedDisruption = len(pods.Items) - minAvailable
	}

	if allowedDisruption < minAllowedDisruption {
		a.Violated = true
		a.Message = fmt.Sprintf("PDB doesnt have enough disruption pod (expect %v, but currently is %v)", minAllowedDisruption, allowedDisruption)
	} else {
		a.Message = fmt.Sprintf("PDB has enough disruption pod (expect %v, currently is %v)", minAllowedDisruption, allowedDisruption)
	}
	return
}

func (p *pdbMinAllowedDisruptionNamespaceRule) GetDelaySeconds(object interface{}) (time.Duration, error) {
	return 0, nil
}
