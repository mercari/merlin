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
		log:        ctrl.Log.WithName("Notifier"),
		scheme:     mgr.GetScheme(),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	if err := notifierReconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	secretUnusedRule := &rules.Cache{}
	hpaInvalidScaleTargetRefRule := &rules.Cache{}
	hpaReplicaPercentageRules := &rules.Cache{}
	namespaceRequiredLabelRules := &rules.Cache{}
	serviceInvalidSelectorRules := &rules.Cache{}
	pdbInvalidSelectorRules := &rules.Cache{}
	pdbMinAllowedDisruptionRules := &rules.Cache{}

	//// resource Reconcilers ////

	if err := (&PodReconciler{
		ResourceReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("Pod"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			resource:      &corev1.Pod{},
			rules:         []*rules.Cache{secretUnusedRule},
		},
	}).SetupWithManager(mgr, func(rawObj runtime.Object) []string {
		obj := rawObj.(*corev1.Pod)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := (&HorizontalPodAutoscalerReconciler{
		ResourceReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("HPA"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			resource:      &autoscalingv1.HorizontalPodAutoscaler{},
			rules: []*rules.Cache{
				hpaReplicaPercentageRules,
				hpaInvalidScaleTargetRefRule,
			},
		},
	}).SetupWithManager(mgr,
		func(rawObj runtime.Object) []string {
			obj := rawObj.(*autoscalingv1.HorizontalPodAutoscaler)
			return []string{obj.Name}
		}); err != nil {
		return err
	}

	if err := (&NamespaceReconciler{
		ResourceReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("Namespace"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			resource:      &corev1.Namespace{},
			rules:         []*rules.Cache{namespaceRequiredLabelRules},
		},
	}).SetupWithManager(mgr, func(rawObj runtime.Object) []string {
		obj := rawObj.(*corev1.Namespace)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := (&PodDisruptionBudgetReconciler{
		ResourceReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("PDB"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			resource:      &policyv1beta1.PodDisruptionBudget{},
			rules: []*rules.Cache{
				pdbMinAllowedDisruptionRules,
				pdbInvalidSelectorRules,
			},
		},
	}).SetupWithManager(mgr, func(rawObj runtime.Object) []string {
		obj := rawObj.(*policyv1beta1.PodDisruptionBudget)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := (&ServiceReconciler{
		ResourceReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("Service"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			resource:      &corev1.Service{},
			rules:         []*rules.Cache{serviceInvalidSelectorRules},
		},
	}).SetupWithManager(mgr, func(rawObj runtime.Object) []string {
		obj := rawObj.(*corev1.Service)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := (&SecretReconciler{
		ResourceReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("Secret"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			resource:      &corev1.Secret{},
			rules:         []*rules.Cache{secretUnusedRule},
		},
	}).SetupWithManager(mgr, func(rawObj runtime.Object) []string {
		obj := rawObj.(*corev1.Secret)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	//// Rule Reconcilers ////

	if err := (&SecretUnusedRuleReconciler{
		RuleReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("SecretUnusedRule"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			rules:         secretUnusedRule,
			ruleFactory:   &rules.SecretUnusedRule{},
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

	if err := (&HPAReplicaPercentageRuleReconciler{
		RuleReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("HPAReplicaPercentageRule"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			rules:         hpaReplicaPercentageRules,
			ruleFactory:   &rules.HPAReplicaPercentageRule{},
		},
	}).SetupWithManager(mgr,
		&merlinv1.ClusterRuleHPAReplicaPercentage{},
		&merlinv1.RuleHPAReplicaPercentage{},
		func(rawObj runtime.Object) []string {
			if clusterRule, ok := rawObj.(*merlinv1.ClusterRuleHPAReplicaPercentage); ok {
				return []string{clusterRule.Name}
			} else if namespaceRule, ok := rawObj.(*merlinv1.RuleHPAReplicaPercentage); ok {
				return []string{namespaceRule.Name}
			}
			return []string{}
		}); err != nil {
		return err
	}

	if err := (&HPAInvalidScaleTargetRefRuleReconciler{
		RuleReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("HPAInvalidScaleTargetRefRule"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			rules:         hpaReplicaPercentageRules,
			ruleFactory:   &rules.HPAInvalidScaleTargetRefRule{},
		},
	}).SetupWithManager(mgr,
		&merlinv1.ClusterRuleHPAInvalidScaleTargetRef{},
		nil,
		func(rawObj runtime.Object) []string {
			rule := rawObj.(*merlinv1.ClusterRuleHPAInvalidScaleTargetRef)
			return []string{rule.Name}
		}); err != nil {
		return err
	}

	if err := (&NamespaceRequiredLabelRuleReconciler{
		RuleReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("NamespaceRequiredLabelRule"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			rules:         namespaceRequiredLabelRules,
			ruleFactory:   &rules.NamespaceRequiredLabelRule{},
		},
	}).SetupWithManager(mgr,
		&merlinv1.ClusterRuleNamespaceRequiredLabel{},
		nil,
		func(rawObj runtime.Object) []string {
			rule := rawObj.(*merlinv1.ClusterRuleNamespaceRequiredLabel)
			return []string{rule.Name}
		}); err != nil {
		return err
	}

	if err := (&ServiceInvalidSelectorRuleReconciler{
		RuleReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("ServiceInvalidSelectorRule"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			rules:         serviceInvalidSelectorRules,
			ruleFactory:   &rules.ServiceInvalidSelectorRule{},
		},
	}).SetupWithManager(mgr,
		&merlinv1.ClusterRuleServiceInvalidSelector{},
		nil,
		func(rawObj runtime.Object) []string {
			rule := rawObj.(*merlinv1.ClusterRuleServiceInvalidSelector)
			return []string{rule.Name}
		}); err != nil {
		return err
	}

	if err := (&PDBInvalidSelectorRuleReconciler{
		RuleReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("PDBInvalidSelectorRule"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			rules:         pdbInvalidSelectorRules,
			ruleFactory:   &rules.PDBInvalidSelectorRule{},
		},
	}).SetupWithManager(mgr,
		&merlinv1.ClusterRulePDBInvalidSelector{},
		nil,
		func(rawObj runtime.Object) []string {
			rule := rawObj.(*merlinv1.ClusterRulePDBInvalidSelector)
			return []string{rule.Name}
		}); err != nil {
		return err
	}

	if err := (&PDBMinAllowedDisruptionRuleReconciler{
		RuleReconciler{
			Client:        mgr.GetClient(),
			log:           ctrl.Log.WithName("PDBMinAllowedDisruptionRule"),
			scheme:        mgr.GetScheme(),
			notifierCache: notifierReconciler.cache,
			rules:         pdbMinAllowedDisruptionRules,
			ruleFactory:   &rules.PDBMinAllowedDisruptionRule{},
		},
	}).SetupWithManager(mgr,
		&merlinv1.ClusterRulePDBMinAllowedDisruption{},
		&merlinv1.RulePDBMinAllowedDisruption{},
		func(rawObj runtime.Object) []string {
			if clusterRule, ok := rawObj.(*merlinv1.ClusterRulePDBMinAllowedDisruption); ok {
				return []string{clusterRule.Name}
			} else if namespaceRule, ok := rawObj.(*merlinv1.RulePDBMinAllowedDisruption); ok {
				return []string{namespaceRule.Name}
			}
			return []string{}
		}); err != nil {
		return err
	}

	return nil
}
