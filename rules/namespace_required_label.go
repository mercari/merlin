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

type NamespaceRequiredLabelRule struct {
	rule
	resource *merlinv1beta1.ClusterRuleNamespaceRequiredLabel
}

func (n *NamespaceRequiredLabelRule) New(ctx context.Context, cli client.Client, logger logr.Logger, key client.ObjectKey) (Rule, error) {
	n.cli = cli
	n.log = logger
	n.status = &Status{}
	n.resource = &merlinv1beta1.ClusterRuleNamespaceRequiredLabel{}
	if err := n.cli.Get(ctx, key, n.resource); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *NamespaceRequiredLabelRule) GetObject() runtime.Object {
	return n.resource
}

func (n NamespaceRequiredLabelRule) GetName() string {
	return strings.Join([]string{getStructName(n.resource), n.resource.Name}, Separator)
}

func (n NamespaceRequiredLabelRule) GetObjectMeta() metav1.ObjectMeta {
	return n.resource.ObjectMeta
}

func (n NamespaceRequiredLabelRule) GetNotification() merlinv1beta1.Notification {
	return n.resource.Spec.Notification
}

func (n *NamespaceRequiredLabelRule) SetFinalizer(finalizer string) {
	n.resource.ObjectMeta.Finalizers = append(n.resource.ObjectMeta.Finalizers, finalizer)
}

func (n *NamespaceRequiredLabelRule) RemoveFinalizer(finalizer string) {
	n.resource.ObjectMeta.Finalizers = removeString(n.resource.ObjectMeta.Finalizers, finalizer)
}

func (n *NamespaceRequiredLabelRule) EvaluateAll(ctx context.Context) (alerts []alert.Alert, err error) {
	namespaceList := &corev1.NamespaceList{}
	if err = n.cli.List(ctx, namespaceList); err != nil {
		return
	}

	if len(namespaceList.Items) == 0 {
		n.log.Info("no namespace found")
		return
	}
	for _, ns := range namespaceList.Items {
		n.log.Info("evaluating", fmt.Sprintf("%T", ns), ns)
		var a alert.Alert
		a, err = n.Evaluate(ctx, &ns)
		if err != nil {
			return
		}
		alerts = append(alerts, a)
	}
	return
}

func (n *NamespaceRequiredLabelRule) Evaluate(ctx context.Context, object interface{}) (a alert.Alert, err error) {
	namespace, ok := object.(*corev1.Namespace)
	if !ok {
		err = fmt.Errorf("unable to convert object to type %T", namespace)
		return
	}
	n.log.Info("evaluating", fmt.Sprintf("%T", namespace), namespace.Name)
	key := client.ObjectKey{Name: namespace.Name}
	a = alert.Alert{
		Suppressed:      n.resource.Spec.Notification.Suppressed,
		Severity:        n.resource.Spec.Notification.Severity,
		MessageTemplate: n.resource.Spec.Notification.CustomMessageTemplate,
		ResourceName:    key.String(),
		ResourceKind:    getStructName(namespace),
		Violated:        false,
	}
	if isStringInSlice(n.resource.Spec.IgnoreNamespaces, key.Name) {
		a.Message = "namespace is ignored by the rule"
		return
	}
	message, err := validateRequiredLabel(n.resource.Spec.Label, namespace.GetLabels())
	if err != nil {
		return
	}
	if message != "" {
		a.Message = message
		a.Violated = true
	}
	n.status.setViolation(key, a.Violated)
	return
}

func (n *NamespaceRequiredLabelRule) GetDelaySeconds(object interface{}) (time.Duration, error) {
	return 0, nil
}
