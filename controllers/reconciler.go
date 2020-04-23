package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/go-logr/logr"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
	"github.com/kouzoh/merlin/notifiers/alert"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type BaseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// Notifiers stores the notifiers as cache, this will be updated when any notifier updates happen,
	// and also servers as cache so we dont need to get list of notifiers every time
	Notifiers *merlinv1.NotifiersCache
	// RuleStatues stores the status of rules, it has sync.Mutex so reconciler process needs to acquire the lock
	// before making changes
	RuleStatues map[string]*RuleStatusWithLock
	// Rules is the list of rules to apply for this reconciler
	Rules []merlinv1.Rule
	// WatchedAPIType is the kubernetes resource type that the reconciler watches.
	WatchedAPIType runtime.Object
}

const FinalizerName = "rule.finalizers.merlin.mercari.com"

func (b *BaseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := b.Log.WithName("Reconcile")
	var rulesToApply []merlinv1.Rule
	var objectKey client.ObjectKey // the object key this reconciler watches, will be empty if the trigger is rules.

	// Check what's been changed - since we watch for both rules and kubernetes resources
	resourceNames := strings.Split(req.Name, Separator)
	if len(resourceNames) >= 2 {
		//  it's clusterRule or rule changes
		l = l.WithValues("rule", req.NamespacedName)
		var rule runtime.Object
		var ruleKind string
		for _, r := range b.Rules {
			if resourceNames[0] == GetStructName(r) {
				ruleKind = GetStructName(r)
				rule = r.DeepCopyObject()
				break
			}
		}

		if rule == nil {
			err := fmt.Errorf("unexpected rule change")
			l.Error(err, req.NamespacedName.String())
			return ctrl.Result{}, err
		}

		if err := b.Client.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: resourceNames[1]}, rule); err != nil {
			if apierrs.IsNotFound(err) {
				l.Info("rule is deleted")
				return ctrl.Result{}, nil
			}
			return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
		}

		r := rule.(merlinv1.Rule)
		if r.GetObjectMeta().DeletionTimestamp.IsZero() {
			if !containsString(r.GetObjectMeta().Finalizers, FinalizerName) {
				l.V(1).Info("Setting finalizer", "finalizer", FinalizerName)
				r.SetFinalizer(FinalizerName)
				if err := b.Update(ctx, r); err != nil {
					l.Error(err, "Failed to set finalizer")
					return ctrl.Result{}, err
				}
			}
		} else if containsString(r.GetObjectMeta().Finalizers, FinalizerName) {
			l.Info("Rule is being delete, clear alerts")
			for _, n := range r.GetNotification().Notifiers {
				notifier, ok := b.Notifiers.Notifiers[n]
				if !ok {
					l.Error(merlinv1.NotifierNotFoundErr, "notifier not found", "notifier", n)
					continue
				}
				l.V(1).Info("removing alert from notifier", "notifier", notifier.Name)
				notifier.ClearRuleAlerts(ruleKind, r.GetName(), "recover alert since rule is being deleted")
				r.RemoveFinalizer(FinalizerName)
				if err := b.Update(ctx, r); err != nil {
					l.Error(err, "Failed to remove finalizer")
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			}
		}
		rulesToApply = []merlinv1.Rule{r}

	} else {
		// resource changes
		object := b.WatchedAPIType.DeepCopyObject()
		objectKey = req.NamespacedName
		l = l.WithValues(GetStructName(b.WatchedAPIType), objectKey)
		if err := b.Client.Get(ctx, objectKey, object); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}

		// get list of applicable rules
		for _, rule := range b.Rules {
			if rule.IsNamespacedRule() {
				// skip namespaced rule since every namespace rule should be in pair with a cluster rule,
				// and if a cluster rule has namespaced rule, its namespaced rule will be checked, so no need to check again.
				continue
			}
			ruleList := rule.List()
			// if a rule has namespaced rule, check namespace rules first,
			//   - if namespace rules exist, just apply namespaced rule
			//   - if no namespace rule exists, get cluster rules to apply
			namespacedRuleList := rule.GetNamespacedRuleList()
			if namespacedRuleList != nil {
				l.V(1).Info("Rule has namespaced rule defined, getting namespaced rules", "namespacedRuleList", GetStructName(namespacedRuleList))
				if err := b.List(ctx, namespacedRuleList, &client.ListOptions{Namespace: req.Namespace}); client.IgnoreNotFound(err) != nil {
					l.Error(err, "failed to list namespaced rules", "rule", GetStructName(rule))
					return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
				}
				for _, r := range namespacedRuleList.ListItems() {
					l.V(1).Info("adding namespaced rule to apply", "rule", r.GetName())
					rulesToApply = append(rulesToApply, r)
				}
			}

			// not namespaced rule or no namespaced rules exists
			if namespacedRuleList == nil || len(namespacedRuleList.ListItems()) <= 0 {
				l.V(1).Info("Rule dosent have namespaced rule defined or none exists, getting cluster rules", "clusterRuleList", GetStructName(ruleList))
				if err := b.List(ctx, ruleList); client.IgnoreNotFound(err) != nil {
					l.Error(err, "failed to list cluster rules", "rule", GetStructName(rule))
					return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
				}
				for _, r := range ruleList.ListItems() {
					namespace := objectKey.Namespace
					if objectKey.Namespace == "" { // for namespace resource its Namespace is empty
						namespace = objectKey.Name
					}
					if !r.IsNamespaceIgnored(namespace) {
						// unlike rule changes - when a resource changes it should be safe to respect ignore namespaces from rules
						l.V(1).Info("adding cluster rule to apply", "rule", r.GetName())
						rulesToApply = append(rulesToApply, r)
					}
				}
			}
		}
	}

	l.V(1).Info("rules to apply", "rules", rulesToApply)
	for _, rule := range rulesToApply {
		b.RuleStatues[rule.GetName()].Lock()

		var isObjectDeleted bool
		var listOptions = &client.ListOptions{}
		resourceList := rule.GetResourceList()
		list := resourceList.List()

		l.V(1).Info("Checking rule/objects changes and list objects", "resource", GetStructName(list))
		if objectKey == (client.ObjectKey{}) {
			l.V(1).Info("rule changes, list objects")
			if rule.IsNamespacedRule() {
				listOptions = rule.GetSelector().AsListOption(req.NamespacedName.Namespace)
			}
			if err := b.Client.List(ctx, list, listOptions); err != nil {
				if apierrs.IsNotFound(err) {
					l.Info("No objects found for evaluation")
					b.RuleStatues[rule.GetName()].Unlock()
					return ctrl.Result{}, nil
				}
				b.RuleStatues[rule.GetName()].Unlock()
				return ctrl.Result{}, err
			}
		} else {
			l.V(1).Info("watched object changes - get only the object")
			if rule.IsNamespacedRule() {
				listOptions = rule.GetSelector().AsListOption(objectKey.Namespace)
			} else {
				listOptions.Namespace = objectKey.Namespace
				listOptions.FieldSelector = fields.Set{metadataNameField: objectKey.Name}.AsSelector()
			}
			if err := b.Client.List(ctx, list, listOptions); client.IgnoreNotFound(err) != nil {
				b.RuleStatues[rule.GetName()].Unlock()
				return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
			}
			if len(resourceList.ListItems()) == 0 {
				l.V(1).Info("no objects found, it's been deleted or ignored by selector")
				isObjectDeleted = true
				resourceList.AddItem(objectKey)
			}
		}

		for _, obj := range resourceList.ListItems() {
			l.V(1).Info("object to validate", "obj", obj)
			newAlert := alert.Alert{
				Suppressed:      rule.GetNotification().Suppressed,
				Severity:        rule.GetNotification().Severity,
				MessageTemplate: rule.GetNotification().CustomMessageTemplate,
				ResourceKind:    GetStructName(obj),
			}

			isViolated := false
			if isObjectDeleted {
				newAlert.Message = "recovered since object is deleted or ignored by rule selector"
				newAlert.ResourceName = objectKey.String()
				rule.SetViolationStatus(objectKey, false)
			} else {
				namespacedName, err := rule.GetObjectNamespacedName(obj)
				if err != nil {
					b.RuleStatues[rule.GetName()].Unlock()
					return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
				}

				if isViolated, newAlert.Message, err = rule.Evaluate(ctx, b.Client, l, obj); err != nil {
					b.RuleStatues[rule.GetName()].Unlock()
					return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
				}

				namespace := namespacedName.Namespace
				if namespace == "" {
					namespace = namespacedName.Name
				}
				if rule.IsNamespaceIgnored(namespace) {
					// update the violation in case the ignore is been added later.
					// can not just ignore the namespaces and not check them
					isViolated = false
					newAlert.Message = fmt.Sprintf("resources in namespace '%s' are ignored", req.Namespace)
				}
				newAlert.ResourceName = namespacedName.String()
				rule.SetViolationStatus(namespacedName, isViolated)
			}

			for _, n := range rule.GetNotification().Notifiers {
				notifier, ok := b.Notifiers.Notifiers[n]
				if !ok {
					l.Error(merlinv1.NotifierNotFoundErr, "notifier not found", "notifier", n)
					continue
				}
				l.V(1).Info("setting alert to notifier", "notifier", notifier.Name, "object", newAlert.ResourceName, "isViolated", isViolated)
				notifier.SetAlert(GetStructName(rule), rule.GetName(), newAlert, isViolated)
			}
		}

		l.V(1).Info("updating rule status", "rule", rule.GetName(), "status", rule.GetStatus())
		if err := b.Status().Update(ctx, rule); err != nil {
			l.Error(err, "unable to update rule status", "rule", rule.GetName())
			b.RuleStatues[rule.GetName()].Unlock()
			return ctrl.Result{RequeueAfter: requeueIntervalForError()}, err
		}
		b.RuleStatues[rule.GetName()].RuleStatus = rule.GetStatus()
		b.RuleStatues[rule.GetName()].Unlock()
	}

	return ctrl.Result{}, nil
}

func requeueIntervalForError() time.Duration {
	rand.Seed(time.Now().UnixNano())
	min := 10
	max := 30
	return time.Duration(rand.Intn(max-min+1)+min) * time.Second
}

func (b *BaseReconciler) SetupWithManager(mgr ctrl.Manager, indexingFunc func(rawObj runtime.Object) []string) error {
	l := b.Log.WithName("SetupWithManager")
	b.RuleStatues = map[string]*RuleStatusWithLock{}

	if err := mgr.GetFieldIndexer().IndexField(b.WatchedAPIType, metadataNameField, indexingFunc); err != nil {
		return err
	}
	builder := ctrl.NewControllerManagedBy(mgr).For(b.WatchedAPIType)

	for _, rule := range b.Rules {
		if err := mgr.GetFieldIndexer().IndexField(rule, metadataNameField, func(rawObj runtime.Object) []string {
			obj := rawObj.(merlinv1.Rule)
			l.Info("indexing", "rule", obj.GetName())
			if _, ok := b.RuleStatues[obj.GetName()]; !ok {
				b.RuleStatues[obj.GetName()] = &RuleStatusWithLock{}
			}
			return []string{obj.GetName()}
		}); err != nil {
			return err
		}
		builder.Watches(&source.Kind{Type: rule}, &EventHandler{Log: l, Kind: GetStructName(rule)})
	}

	l.Info("initialize manager", "watch for resource", GetStructName(b.WatchedAPIType))
	return builder.WithEventFilter(&EventFilter{Log: l}).
		Named(GetStructName(b.WatchedAPIType)).
		Complete(b)
}
