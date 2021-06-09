package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	merlinv1beta1 "github.com/mercari/merlin/api/v1beta1"
	"github.com/mercari/merlin/rules"
)

var notifierReconciler *NotifierReconciler

func SetupReconcilers(mgr manager.Manager) error {

	alertMetrics := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: fmt.Sprintf("merlin_violation"),
			Help: "Merlin - indicates if a resource has violated rules (gauge)",
		},
		[]string{"rule", "rule_name", "resource_name", "resource_namespace", "resource_kind"},
	)
	metrics.Registry.MustRegister(alertMetrics)

	notifierReconciler = &NotifierReconciler{
		Client:     mgr.GetClient(),
		log:        ctrl.Log.WithName("Notifier"),
		scheme:     mgr.GetScheme(),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	if err := notifierReconciler.SetupWithManager(mgr, alertMetrics); err != nil {
		return err
	}

	secretUnusedRule := &rulesCache{}
	configMapUnusedRule := &rulesCache{}
	hpaInvalidScaleTargetRefRule := &rulesCache{}
	hpaReplicaPercentageRules := &rulesCache{}
	namespaceRequiredLabelRules := &rulesCache{}
	serviceInvalidSelectorRules := &rulesCache{}
	pdbInvalidSelectorRules := &rulesCache{}
	pdbMinAllowedDisruptionRules := &rulesCache{}

	//// resource Reconcilers ////

	if err := (&PodReconciler{
		ResourceReconciler{
			Client:    mgr.GetClient(),
			log:       ctrl.Log.WithName("Pod"),
			scheme:    mgr.GetScheme(),
			notifiers: notifierReconciler.cache,
			resource:  &corev1.Pod{},
			rules:     []*rulesCache{secretUnusedRule, configMapUnusedRule},
		},
	}).SetupWithManager(mgr, func(rawObj runtime.Object) []string {
		obj := rawObj.(*corev1.Pod)
		return []string{obj.Name}
	}); err != nil {
		return err
	}

	if err := (&HorizontalPodAutoscalerReconciler{
		ResourceReconciler{
			Client:    mgr.GetClient(),
			log:       ctrl.Log.WithName("HPA"),
			scheme:    mgr.GetScheme(),
			notifiers: notifierReconciler.cache,
			resource:  &autoscalingv1.HorizontalPodAutoscaler{},
			rules: []*rulesCache{
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
			Client:    mgr.GetClient(),
			log:       ctrl.Log.WithName("Namespace"),
			scheme:    mgr.GetScheme(),
			notifiers: notifierReconciler.cache,
			resource:  &corev1.Namespace{},
			rules:     []*rulesCache{namespaceRequiredLabelRules},
		},
	}).SetupWithManager(mgr,
		func(rawObj runtime.Object) []string {
			obj := rawObj.(*corev1.Namespace)
			return []string{obj.Name}
		}); err != nil {
		return err
	}

	if err := (&PodDisruptionBudgetReconciler{
		ResourceReconciler{
			Client:    mgr.GetClient(),
			log:       ctrl.Log.WithName("PDB"),
			scheme:    mgr.GetScheme(),
			notifiers: notifierReconciler.cache,
			resource:  &policyv1beta1.PodDisruptionBudget{},
			rules: []*rulesCache{
				pdbMinAllowedDisruptionRules,
				pdbInvalidSelectorRules,
			},
		},
	}).SetupWithManager(mgr,
		func(rawObj runtime.Object) []string {
			obj := rawObj.(*policyv1beta1.PodDisruptionBudget)
			return []string{obj.Name}
		}); err != nil {
		return err
	}

	if err := (&ServiceReconciler{
		ResourceReconciler{
			Client:    mgr.GetClient(),
			log:       ctrl.Log.WithName("Service"),
			scheme:    mgr.GetScheme(),
			notifiers: notifierReconciler.cache,
			resource:  &corev1.Service{},
			rules:     []*rulesCache{serviceInvalidSelectorRules},
		},
	}).SetupWithManager(mgr,
		func(rawObj runtime.Object) []string {
			obj := rawObj.(*corev1.Service)
			return []string{obj.Name}
		}); err != nil {
		return err
	}

	if err := (&SecretReconciler{
		ResourceReconciler{
			Client:    mgr.GetClient(),
			log:       ctrl.Log.WithName("Secret"),
			scheme:    mgr.GetScheme(),
			notifiers: notifierReconciler.cache,
			resource:  &corev1.Secret{},
			rules:     []*rulesCache{secretUnusedRule},
		},
	}).SetupWithManager(mgr,
		func(rawObj runtime.Object) []string {
			obj := rawObj.(*corev1.Secret)
			return []string{obj.Name}
		}); err != nil {
		return err
	}

	if err := (&ConfigMapReconciler{
		ResourceReconciler{
			Client:    mgr.GetClient(),
			log:       ctrl.Log.WithName("ConfigMap"),
			scheme:    mgr.GetScheme(),
			notifiers: notifierReconciler.cache,
			resource:  &corev1.ConfigMap{},
			rules:     []*rulesCache{configMapUnusedRule},
		},
	}).SetupWithManager(mgr,
		func(rawObj runtime.Object) []string {
			obj := rawObj.(*corev1.ConfigMap)
			return []string{obj.Name}
		}); err != nil {
		return err
	}

	//// Rule Reconcilers ////

	if err := (&SecretUnusedRuleReconciler{
		RuleReconciler{
			Client:      mgr.GetClient(),
			log:         ctrl.Log.WithName("SecretUnusedRule"),
			scheme:      mgr.GetScheme(),
			notifiers:   notifierReconciler.cache,
			rules:       secretUnusedRule,
			ruleFactory: &rules.SecretUnusedRule{},
		},
	}).SetupWithManager(mgr,
		alertMetrics,
		&merlinv1beta1.ClusterRuleSecretUnused{},
		nil,
		func(rawObj runtime.Object) []string {
			obj := rawObj.(*merlinv1beta1.ClusterRuleSecretUnused)
			return []string{obj.ObjectMeta.Name}
		}); err != nil {
		return err
	}

	if err := (&ConfigMapUnusedRuleReconciler{
		RuleReconciler{
			Client:      mgr.GetClient(),
			log:         ctrl.Log.WithName("ConfigMapUnusedRule"),
			scheme:      mgr.GetScheme(),
			notifiers:   notifierReconciler.cache,
			rules:       secretUnusedRule,
			ruleFactory: &rules.SecretUnusedRule{},
		},
	}).SetupWithManager(mgr,
		alertMetrics,
		&merlinv1beta1.ClusterRuleConfigMapUnused{},
		nil,
		func(rawObj runtime.Object) []string {
			obj := rawObj.(*merlinv1beta1.ClusterRuleConfigMapUnused)
			return []string{obj.ObjectMeta.Name}
		}); err != nil {
		return err
	}

	if err := (&HPAReplicaPercentageRuleReconciler{
		RuleReconciler{
			Client:      mgr.GetClient(),
			log:         ctrl.Log.WithName("HPAReplicaPercentageRule"),
			scheme:      mgr.GetScheme(),
			notifiers:   notifierReconciler.cache,
			rules:       hpaReplicaPercentageRules,
			ruleFactory: &rules.HPAReplicaPercentageRule{},
		},
	}).SetupWithManager(mgr,
		alertMetrics,
		&merlinv1beta1.ClusterRuleHPAReplicaPercentage{},
		&merlinv1beta1.RuleHPAReplicaPercentage{},
		func(rawObj runtime.Object) []string {
			if clusterRule, ok := rawObj.(*merlinv1beta1.ClusterRuleHPAReplicaPercentage); ok {
				return []string{clusterRule.Name}
			} else if namespaceRule, ok := rawObj.(*merlinv1beta1.RuleHPAReplicaPercentage); ok {
				return []string{namespaceRule.Name}
			}
			return []string{}
		}); err != nil {
		return err
	}

	if err := (&HPAInvalidScaleTargetRefRuleReconciler{
		RuleReconciler{
			Client:      mgr.GetClient(),
			log:         ctrl.Log.WithName("HPAInvalidScaleTargetRefRule"),
			scheme:      mgr.GetScheme(),
			notifiers:   notifierReconciler.cache,
			rules:       hpaReplicaPercentageRules,
			ruleFactory: &rules.HPAInvalidScaleTargetRefRule{},
		},
	}).SetupWithManager(mgr,
		alertMetrics,
		&merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef{},
		nil,
		func(rawObj runtime.Object) []string {
			rule := rawObj.(*merlinv1beta1.ClusterRuleHPAInvalidScaleTargetRef)
			return []string{rule.Name}
		}); err != nil {
		return err
	}

	if err := (&NamespaceRequiredLabelRuleReconciler{
		RuleReconciler{
			Client:      mgr.GetClient(),
			log:         ctrl.Log.WithName("NamespaceRequiredLabelRule"),
			scheme:      mgr.GetScheme(),
			notifiers:   notifierReconciler.cache,
			rules:       namespaceRequiredLabelRules,
			ruleFactory: &rules.NamespaceRequiredLabelRule{},
		},
	}).SetupWithManager(mgr,
		alertMetrics,
		&merlinv1beta1.ClusterRuleNamespaceRequiredLabel{},
		nil,
		func(rawObj runtime.Object) []string {
			rule := rawObj.(*merlinv1beta1.ClusterRuleNamespaceRequiredLabel)
			return []string{rule.Name}
		}); err != nil {
		return err
	}

	if err := (&ServiceInvalidSelectorRuleReconciler{
		RuleReconciler{
			Client:      mgr.GetClient(),
			log:         ctrl.Log.WithName("ServiceInvalidSelectorRule"),
			scheme:      mgr.GetScheme(),
			notifiers:   notifierReconciler.cache,
			rules:       serviceInvalidSelectorRules,
			ruleFactory: &rules.ServiceInvalidSelectorRule{},
		},
	}).SetupWithManager(mgr,
		alertMetrics,
		&merlinv1beta1.ClusterRuleServiceInvalidSelector{},
		nil,
		func(rawObj runtime.Object) []string {
			rule := rawObj.(*merlinv1beta1.ClusterRuleServiceInvalidSelector)
			return []string{rule.Name}
		}); err != nil {
		return err
	}

	if err := (&PDBInvalidSelectorRuleReconciler{
		RuleReconciler{
			Client:      mgr.GetClient(),
			log:         ctrl.Log.WithName("PDBInvalidSelectorRule"),
			scheme:      mgr.GetScheme(),
			notifiers:   notifierReconciler.cache,
			rules:       pdbInvalidSelectorRules,
			ruleFactory: &rules.PDBInvalidSelectorRule{},
		},
	}).SetupWithManager(mgr,
		alertMetrics,
		&merlinv1beta1.ClusterRulePDBInvalidSelector{},
		nil,
		func(rawObj runtime.Object) []string {
			rule := rawObj.(*merlinv1beta1.ClusterRulePDBInvalidSelector)
			return []string{rule.Name}
		}); err != nil {
		return err
	}

	if err := (&PDBMinAllowedDisruptionRuleReconciler{
		RuleReconciler{
			Client:      mgr.GetClient(),
			log:         ctrl.Log.WithName("PDBMinAllowedDisruptionRule"),
			scheme:      mgr.GetScheme(),
			notifiers:   notifierReconciler.cache,
			rules:       pdbMinAllowedDisruptionRules,
			ruleFactory: &rules.PDBMinAllowedDisruptionRule{},
		},
	}).SetupWithManager(mgr,
		alertMetrics,
		&merlinv1beta1.ClusterRulePDBMinAllowedDisruption{},
		&merlinv1beta1.RulePDBMinAllowedDisruption{},
		func(rawObj runtime.Object) []string {
			if clusterRule, ok := rawObj.(*merlinv1beta1.ClusterRulePDBMinAllowedDisruption); ok {
				return []string{clusterRule.Name}
			} else if namespaceRule, ok := rawObj.(*merlinv1beta1.RulePDBMinAllowedDisruption); ok {
				return []string{namespaceRule.Name}
			}
			return []string{}
		}); err != nil {
		return err
	}

	return nil
}
