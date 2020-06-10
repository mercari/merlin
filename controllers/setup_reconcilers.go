package controllers

import (
	"net/http"
	"time"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	merlinv1 "github.com/kouzoh/merlin/api/v1"
	"github.com/kouzoh/merlin/rules"
)

var notifierReconciler *NotifierReconciler

func SetupReconcilers(mgr manager.Manager) error {
	notifierReconciler = &NotifierReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("Notifier"),
		Scheme:     mgr.GetScheme(),
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
	if err := notifierReconciler.SetupWithManager(mgr); err != nil {
		return err
	}
	secretUnusedRule := rules.NewSecretUnusedRule(mgr.GetClient(), ctrl.Log)

	if err := (&SecretUnusedRuleReconciler{
		RuleReconciler{
			Client:        mgr.GetClient(),
			Log:           ctrl.Log.WithName("SecretUnusedRule"),
			Scheme:        mgr.GetScheme(),
			NotifierCache: notifierReconciler.Cache,
			Rule:          secretUnusedRule,
		},
	}).SetupWithManager(mgr,
		&merlinv1.ClusterRuleSecretUnused{},
		nil,
		func(rawObj runtime.Object) []string {
			obj := rawObj.(*merlinv1.ClusterRuleSecretUnused)
			return []string{obj.ObjectMeta.Name}
		}); err != nil {
		return err
	}

	if err := (&PodReconciler{
		ResourceReconciler{
			Client:        mgr.GetClient(),
			Log:           ctrl.Log.WithName("Pod"),
			Scheme:        mgr.GetScheme(),
			NotifierCache: notifierReconciler.Cache,
			Rules:         []rules.Rule{secretUnusedRule},
			Resource:      &corev1.Pod{},
		},
	}).SetupWithManager(mgr, func(rawObj runtime.Object) []string {
		obj := rawObj.(*corev1.Pod)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := (&HorizontalPodAutoscalerReconciler{
		BaseReconciler{
			Client:    mgr.GetClient(),
			Log:       ctrl.Log.WithName("HPA"),
			Scheme:    mgr.GetScheme(),
			Notifiers: notifierReconciler.Cache,
			Rules: []merlinv1.Rule{
				&merlinv1.ClusterRuleHPAInvalidScaleTargetRef{},
				&merlinv1.ClusterRuleHPAReplicaPercentage{},
				&merlinv1.RuleHPAReplicaPercentage{},
			},
			WatchedAPIType: &autoscalingv1.HorizontalPodAutoscaler{},
		},
	}).SetupWithManager(mgr, func(rawObj runtime.Object) []string {
		obj := rawObj.(*autoscalingv1.HorizontalPodAutoscaler)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := (&NamespaceReconciler{
		BaseReconciler{
			Client:    mgr.GetClient(),
			Log:       ctrl.Log.WithName("Namespace"),
			Scheme:    mgr.GetScheme(),
			Notifiers: notifierReconciler.Cache,
			Rules: []merlinv1.Rule{
				&merlinv1.ClusterRuleNamespaceRequiredLabel{},
			},
			WatchedAPIType: &corev1.Namespace{},
		},
	}).SetupWithManager(mgr, func(rawObj runtime.Object) []string {
		obj := rawObj.(*corev1.Namespace)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := (&PodDisruptionBudgetReconciler{
		BaseReconciler{
			Client:    mgr.GetClient(),
			Log:       ctrl.Log.WithName("PDB"),
			Scheme:    mgr.GetScheme(),
			Notifiers: notifierReconciler.Cache,
			Rules: []merlinv1.Rule{
				&merlinv1.ClusterRulePDBInvalidSelector{},
				&merlinv1.ClusterRulePDBMinAllowedDisruption{},
				&merlinv1.RulePDBMinAllowedDisruption{},
			},
			WatchedAPIType: &policyv1beta1.PodDisruptionBudget{},
		},
	}).SetupWithManager(mgr, func(rawObj runtime.Object) []string {
		obj := rawObj.(*policyv1beta1.PodDisruptionBudget)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := (&ServiceReconciler{
		BaseReconciler{
			Client:    mgr.GetClient(),
			Log:       ctrl.Log.WithName("Service"),
			Scheme:    mgr.GetScheme(),
			Notifiers: notifierReconciler.Cache,
			Rules: []merlinv1.Rule{
				&merlinv1.ClusterRuleServiceInvalidSelector{},
			},
			WatchedAPIType: &corev1.Service{},
		},
	}).SetupWithManager(mgr, func(rawObj runtime.Object) []string {
		obj := rawObj.(*corev1.Service)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	return nil
}
