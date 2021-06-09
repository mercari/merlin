package rules

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mercari/merlin/alert"
	merlinv1beta1 "github.com/mercari/merlin/api/v1beta1"
	"github.com/mercari/merlin/mocks"
)

func Test_PDBMinAllowedDisruptionRule_Basic(t *testing.T) {
	notification := merlinv1beta1.Notification{
		Notifiers:  []string{"testNotifier"},
		Suppressed: true,
	}

	cases := []struct {
		desc       string
		ruleName   string
		rule       Rule
		objectMeta metav1.ObjectMeta
	}{
		{
			desc:       "clusterRule",
			objectMeta: metav1.ObjectMeta{Name: "test-r"},
			ruleName:   "ClusterRulePDBMinAllowedDisruption/test-r",
			rule: &pdbMinAllowedDisruptionClusterRule{
				resource: &merlinv1beta1.ClusterRulePDBMinAllowedDisruption{
					ObjectMeta: metav1.ObjectMeta{Name: "test-r"},
					Spec: merlinv1beta1.ClusterRulePDBMinAllowedDisruptionSpec{
						Notification: notification,
					},
				},
			},
		},
		{
			desc:       "namespaceRule",
			objectMeta: metav1.ObjectMeta{Name: "test-r"},
			ruleName:   "RulePDBMinAllowedDisruption/test-r",
			rule: &pdbMinAllowedDisruptionNamespaceRule{
				resource: &merlinv1beta1.RulePDBMinAllowedDisruption{
					ObjectMeta: metav1.ObjectMeta{Name: "test-r"},
					Spec: merlinv1beta1.RulePDBMinAllowedDisruptionSpec{
						Notification: notification,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			assert.Equal(t, tc.objectMeta, tc.rule.GetObjectMeta())
			assert.Equal(t, notification, tc.rule.GetNotification())
			assert.Equal(t, tc.ruleName, tc.rule.GetName())
			finalizer := "test.finalizer"
			tc.rule.SetFinalizer(finalizer)
			assert.Equal(t, finalizer, tc.rule.GetObjectMeta().Finalizers[0])
			tc.rule.RemoveFinalizer(finalizer)
			assert.Empty(t, tc.rule.GetObjectMeta().Finalizers)
		})
	}
}

func Test_PDBMinAllowedDisruptionRule_NewRule(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)

	cases := []struct {
		desc        string
		ruleFactory RuleFactory
		key         client.ObjectKey
		mockCall    func(client.ObjectKey) runtime.Object
	}{
		{
			desc:        "clusterRule",
			key:         client.ObjectKey{Namespace: "", Name: "test-rule"},
			ruleFactory: &PDBMinAllowedDisruptionRule{},
			mockCall: func(key client.ObjectKey) runtime.Object {
				merlinRule := merlinv1beta1.ClusterRulePDBMinAllowedDisruption{
					ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name},
					Spec: merlinv1beta1.ClusterRulePDBMinAllowedDisruptionSpec{
						Notification: merlinv1beta1.Notification{
							Notifiers:  []string{"testNotifier"},
							Suppressed: true,
						},
					},
				}
				mockClient.EXPECT().
					Get(ctx, key, &merlinv1beta1.ClusterRulePDBMinAllowedDisruption{}).
					SetArg(2, merlinRule).
					Return(nil).
					Times(1)
				return &merlinRule
			},
		},
		{
			desc:        "namespaceRule",
			key:         client.ObjectKey{Namespace: "test-ns", Name: "test-rule"},
			ruleFactory: &PDBMinAllowedDisruptionRule{},
			mockCall: func(key client.ObjectKey) runtime.Object {
				merlinRule := merlinv1beta1.RulePDBMinAllowedDisruption{
					ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name},
					Spec: merlinv1beta1.RulePDBMinAllowedDisruptionSpec{
						Notification: merlinv1beta1.Notification{
							Notifiers:  []string{"testNotifier"},
							Suppressed: true,
						},
					},
				}
				mockClient.EXPECT().
					Get(ctx, key, &merlinv1beta1.RulePDBMinAllowedDisruption{}).
					SetArg(2, merlinRule).
					Return(nil).
					Times(1)
				return &merlinRule
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			merlinRule := tc.mockCall(tc.key)
			r, err := tc.ruleFactory.New(ctx, mockClient, log, tc.key)
			assert.NoError(tt, err)
			assert.Equal(tt, merlinRule, r.GetObject())
			delay, err := r.GetDelaySeconds(&policyv1beta1.PodDisruptionBudget{})
			assert.NoError(tt, err)
			assert.Equal(tt, time.Duration(0), delay)
		})
	}
}

func Test_PDBMinAllowedDisruptionRule_Evaluate(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	ruleFactory := &PDBMinAllowedDisruptionRule{}
	notification := merlinv1beta1.Notification{Notifiers: []string{"testNotifier"}}
	clusterRuleKey := client.ObjectKey{Namespace: "", Name: "clusterRule"}
	namespaceRuleKey := client.ObjectKey{Namespace: "testNS", Name: "namespaceRule"}

	cases := []struct {
		desc      string
		key       client.ObjectKey
		mockCalls []*gomock.Call
		expect    alert.Alert
		resource  interface{}
		expectErr bool
	}{
		{
			desc: "clusterRule - non pdb should return err",
			key:  clusterRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().Get(ctx, clusterRuleKey, &merlinv1beta1.ClusterRulePDBMinAllowedDisruption{}).Return(nil),
			},
			resource:  "non-pdb",
			expectErr: true,
		},
		{
			desc: "namespaceRule - non pdb should return err",
			key:  namespaceRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, namespaceRuleKey, &merlinv1beta1.RulePDBMinAllowedDisruption{}).
					Return(nil),
			},
			resource:  "non-pdb",
			expectErr: true,
		},
		{
			desc: "clusterRule - ignored namespace's pdb should not return violated alert",
			key:  clusterRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, clusterRuleKey, &merlinv1beta1.ClusterRulePDBMinAllowedDisruption{}).
					SetArg(2, merlinv1beta1.ClusterRulePDBMinAllowedDisruption{
						Spec: merlinv1beta1.ClusterRulePDBMinAllowedDisruptionSpec{
							IgnoreNamespaces: []string{"ignoredNS"},
							Notification:     notification,
						},
					}).
					Return(nil),
			},
			resource: &policyv1beta1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{Name: "testPDB", Namespace: "ignoredNS"},
				Spec:       policyv1beta1.PodDisruptionBudgetSpec{MaxUnavailable: &intstr.IntOrString{IntVal: 2}},
			},
			expect: alert.Alert{
				Message:      "namespace is ignored by the rule",
				ResourceKind: "PodDisruptionBudget",
				ResourceName: "ignoredNS/testPDB",
			},
		},
		{
			desc: "clusterRule - not enough pods with MaxUnavailable set should return violated alert",
			key:  clusterRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, clusterRuleKey, &merlinv1beta1.ClusterRulePDBMinAllowedDisruption{}).
					SetArg(2, merlinv1beta1.ClusterRulePDBMinAllowedDisruption{
						Spec: merlinv1beta1.ClusterRulePDBMinAllowedDisruptionSpec{
							Notification:         notification,
							MinAllowedDisruption: 2,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     "test",
						LabelSelector: labels.Set(map[string]string{"app": "test"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Labels: map[string]string{"app": "test"}}},
						},
					}).
					Return(nil),
			},
			resource: &policyv1beta1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{Name: "testPDB", Namespace: "test"},
				Spec: policyv1beta1.PodDisruptionBudgetSpec{
					MaxUnavailable: &intstr.IntOrString{IntVal: 1},
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					}},
			},
			expect: alert.Alert{
				Message:      "PDB doesnt have enough disruption pod (expect 2, but currently is 1)",
				ResourceKind: "PodDisruptionBudget",
				ResourceName: "test/testPDB",
				Violated:     true,
			},
		},
		{
			desc: "namespaceRule - not enough pods with MaxUnavailable set should return violated alert",
			key:  namespaceRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, namespaceRuleKey, &merlinv1beta1.RulePDBMinAllowedDisruption{}).
					SetArg(2, merlinv1beta1.RulePDBMinAllowedDisruption{
						Spec: merlinv1beta1.RulePDBMinAllowedDisruptionSpec{
							Notification:         notification,
							MinAllowedDisruption: 2,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     "test",
						LabelSelector: labels.Set(map[string]string{"app": "test"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Labels: map[string]string{"app": "test"}}},
						},
					}).
					Return(nil),
			},
			resource: &policyv1beta1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{Name: "testPDB", Namespace: "test"},
				Spec: policyv1beta1.PodDisruptionBudgetSpec{
					MaxUnavailable: &intstr.IntOrString{IntVal: 1},
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					}},
			},
			expect: alert.Alert{
				Message:      "PDB doesnt have enough disruption pod (expect 2, but currently is 1)",
				ResourceKind: "PodDisruptionBudget",
				ResourceName: "test/testPDB",
				Violated:     true,
			},
		},
		{
			desc: "clusterRule - not enough pods with MinAvailable set should return violated alert",
			key:  clusterRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, clusterRuleKey, &merlinv1beta1.ClusterRulePDBMinAllowedDisruption{}).
					SetArg(2, merlinv1beta1.ClusterRulePDBMinAllowedDisruption{
						Spec: merlinv1beta1.ClusterRulePDBMinAllowedDisruptionSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     "test",
						LabelSelector: labels.Set(map[string]string{"app": "test"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Labels: map[string]string{"app": "test"}}},
						},
					}).
					Return(nil),
			},
			resource: &policyv1beta1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{Name: "testPDB", Namespace: "test"},
				Spec: policyv1beta1.PodDisruptionBudgetSpec{
					MinAvailable: &intstr.IntOrString{IntVal: 1},
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					}},
			},
			expect: alert.Alert{
				Message:      "PDB doesnt have enough disruption pod (expect 1, but currently is 0)",
				ResourceKind: "PodDisruptionBudget",
				ResourceName: "test/testPDB",
				Violated:     true,
			},
		},
		{
			desc: "namespaceRule - not enough pods with MinAvailable set should return violated alert",
			key:  namespaceRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, namespaceRuleKey, &merlinv1beta1.RulePDBMinAllowedDisruption{}).
					SetArg(2, merlinv1beta1.RulePDBMinAllowedDisruption{
						Spec: merlinv1beta1.RulePDBMinAllowedDisruptionSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     "test",
						LabelSelector: labels.Set(map[string]string{"app": "test"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Labels: map[string]string{"app": "test"}}},
						},
					}).
					Return(nil),
			},
			resource: &policyv1beta1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{Name: "testPDB", Namespace: "test"},
				Spec: policyv1beta1.PodDisruptionBudgetSpec{
					MinAvailable: &intstr.IntOrString{IntVal: 1},
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					}},
			},
			expect: alert.Alert{
				Message:      "PDB doesnt have enough disruption pod (expect 1, but currently is 0)",
				ResourceKind: "PodDisruptionBudget",
				ResourceName: "test/testPDB",
				Violated:     true,
			},
		},
		{
			desc: "clusterRule - enough pods should not return violated alert",
			key:  clusterRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, clusterRuleKey, &merlinv1beta1.ClusterRulePDBMinAllowedDisruption{}).
					SetArg(2, merlinv1beta1.ClusterRulePDBMinAllowedDisruption{
						Spec: merlinv1beta1.ClusterRulePDBMinAllowedDisruptionSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     "test",
						LabelSelector: labels.Set(map[string]string{"app": "test"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Labels: map[string]string{"app": "test"}}},
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Labels: map[string]string{"app": "test2"}}},
						},
					}).
					Return(nil),
			},
			resource: &policyv1beta1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{Name: "testPDB", Namespace: "test"},
				Spec: policyv1beta1.PodDisruptionBudgetSpec{
					MinAvailable: &intstr.IntOrString{IntVal: 1},
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					}},
			},
			expect: alert.Alert{
				Message:      "PDB has enough disruption pod (expect 1, currently is 1)",
				ResourceKind: "PodDisruptionBudget",
				ResourceName: "test/testPDB",
			},
		},
		{
			desc: "namespaceRule - enough pods should not return violated alert",
			key:  namespaceRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, namespaceRuleKey, &merlinv1beta1.RulePDBMinAllowedDisruption{}).
					SetArg(2, merlinv1beta1.RulePDBMinAllowedDisruption{
						Spec: merlinv1beta1.RulePDBMinAllowedDisruptionSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     "test",
						LabelSelector: labels.Set(map[string]string{"app": "test"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Labels: map[string]string{"app": "test"}}},
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Labels: map[string]string{"app": "test2"}}},
						},
					}).
					Return(nil),
			},
			resource: &policyv1beta1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{Name: "testPDB", Namespace: "test"},
				Spec: policyv1beta1.PodDisruptionBudgetSpec{
					MinAvailable: &intstr.IntOrString{IntVal: 1},
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					}},
			},
			expect: alert.Alert{
				Message:      "PDB has enough disruption pod (expect 1, currently is 1)",
				ResourceKind: "PodDisruptionBudget",
				ResourceName: "test/testPDB",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			for _, call := range tc.mockCalls {
				call.Times(1)
			}
			r, err := ruleFactory.New(ctx, mockClient, log, tc.key)
			assert.NoError(tt, err)

			a, err := r.Evaluate(ctx, tc.resource)
			if tc.expectErr {
				assert.Error(tt, err)
			} else {
				assert.NoError(tt, err)
				assert.Equal(tt, tc.expect, a)
			}
		})
	}
}

func Test_PDBMinAllowedDisruptionRule_EvaluateAll(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	ruleFactory := &PDBMinAllowedDisruptionRule{}
	notification := merlinv1beta1.Notification{Notifiers: []string{"testNotifier"}}
	clusterRuleKey := client.ObjectKey{Namespace: "", Name: "clusterRule"}
	namespaceRuleKey := client.ObjectKey{Namespace: "testNS", Name: "namespaceRule"}

	cases := []struct {
		desc      string
		key       client.ObjectKey
		mockCalls []*gomock.Call
		expect    []alert.Alert
		expectErr bool
	}{
		{
			desc: "clusterRule - no pdb returns nil alerts",
			key:  clusterRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, clusterRuleKey, &merlinv1beta1.ClusterRulePDBMinAllowedDisruption{}).
					SetArg(2, merlinv1beta1.ClusterRulePDBMinAllowedDisruption{
						ObjectMeta: metav1.ObjectMeta{Namespace: clusterRuleKey.Namespace, Name: "cRule"},
						Spec: merlinv1beta1.ClusterRulePDBMinAllowedDisruptionSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &policyv1beta1.PodDisruptionBudgetList{}).
					Return(nil),
			},
		},
		{
			desc: "namespaceRule - no pdb returns nil alerts",
			key:  namespaceRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, namespaceRuleKey, &merlinv1beta1.RulePDBMinAllowedDisruption{}).
					SetArg(2, merlinv1beta1.RulePDBMinAllowedDisruption{
						ObjectMeta: metav1.ObjectMeta{Namespace: namespaceRuleKey.Namespace, Name: "nsRule"},
						Spec: merlinv1beta1.RulePDBMinAllowedDisruptionSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &policyv1beta1.PodDisruptionBudgetList{},
						&client.ListOptions{Namespace: namespaceRuleKey.Namespace}).
					Return(nil),
			},
		},
		{
			desc: "clusterRule - violated pdb should have alert violated to true",
			key:  clusterRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, clusterRuleKey, &merlinv1beta1.ClusterRulePDBMinAllowedDisruption{}).
					SetArg(2, merlinv1beta1.ClusterRulePDBMinAllowedDisruption{
						Spec: merlinv1beta1.ClusterRulePDBMinAllowedDisruptionSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &policyv1beta1.PodDisruptionBudgetList{}).
					SetArg(1, policyv1beta1.PodDisruptionBudgetList{
						Items: []policyv1beta1.PodDisruptionBudget{
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pdb1"},
								Spec: policyv1beta1.PodDisruptionBudgetSpec{
									MinAvailable: &intstr.IntOrString{IntVal: 1},
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{"app": "test"},
									},
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "test2", Name: "pdb2"},
								Spec: policyv1beta1.PodDisruptionBudgetSpec{
									MinAvailable: &intstr.IntOrString{IntVal: 2},
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{"app": "test2"},
									},
								},
							},
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     "test",
						LabelSelector: labels.Set(map[string]string{"app": "test"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Labels: map[string]string{"app": "test"}}},
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod2", Labels: map[string]string{"app": "test"}}},
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     "test2",
						LabelSelector: labels.Set(map[string]string{"app": "test2"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod3", Labels: map[string]string{"app": "test2"}}},
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod4", Labels: map[string]string{"app": "test2"}}},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Message:      "PDB has enough disruption pod (expect 1, currently is 1)",
					ResourceKind: "PodDisruptionBudget",
					ResourceName: "test/pdb1"},
				{
					Message:      "PDB doesnt have enough disruption pod (expect 1, but currently is 0)",
					ResourceKind: "PodDisruptionBudget",
					ResourceName: "test2/pdb2",
					Violated:     true,
				},
			},
		},
		{
			desc: "namespaceRule - violated pdb should have alert violated to true",
			key:  namespaceRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, namespaceRuleKey, &merlinv1beta1.RulePDBMinAllowedDisruption{}).
					SetArg(2, merlinv1beta1.RulePDBMinAllowedDisruption{
						ObjectMeta: metav1.ObjectMeta{Namespace: namespaceRuleKey.Namespace, Name: "nsRule"},
						Spec: merlinv1beta1.RulePDBMinAllowedDisruptionSpec{
							Notification: notification,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &policyv1beta1.PodDisruptionBudgetList{},
						&client.ListOptions{Namespace: namespaceRuleKey.Namespace}).
					SetArg(1, policyv1beta1.PodDisruptionBudgetList{
						Items: []policyv1beta1.PodDisruptionBudget{
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: namespaceRuleKey.Namespace, Name: "pdb1"},
								Spec: policyv1beta1.PodDisruptionBudgetSpec{
									MinAvailable: &intstr.IntOrString{IntVal: 1},
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{"app": "test"},
									},
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: namespaceRuleKey.Namespace, Name: "pdb2"},
								Spec: policyv1beta1.PodDisruptionBudgetSpec{
									MinAvailable: &intstr.IntOrString{IntVal: 2},
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{"app": "test2"},
									},
								},
							},
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     namespaceRuleKey.Namespace,
						LabelSelector: labels.Set(map[string]string{"app": "test"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Labels: map[string]string{"app": "test"}}},
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod2", Labels: map[string]string{"app": "test"}}},
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{
						Namespace:     namespaceRuleKey.Namespace,
						LabelSelector: labels.Set(map[string]string{"app": "test2"}).AsSelector()}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod3", Labels: map[string]string{"app": "test2"}}},
							{ObjectMeta: metav1.ObjectMeta{Name: "test-pod4", Labels: map[string]string{"app": "test2"}}},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Message:      "PDB has enough disruption pod (expect 1, currently is 1)",
					ResourceKind: "PodDisruptionBudget",
					ResourceName: "testNS/pdb1"},
				{
					Message:      "PDB doesnt have enough disruption pod (expect 1, but currently is 0)",
					ResourceKind: "PodDisruptionBudget",
					ResourceName: "testNS/pdb2",
					Violated:     true,
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			for _, call := range tc.mockCalls {
				call.Times(1)
			}
			r, err := ruleFactory.New(ctx, mockClient, log, tc.key)
			assert.NoError(tt, err)

			alerts, err := r.EvaluateAll(ctx)
			if tc.expectErr {
				assert.Error(tt, err)
			} else {
				assert.NoError(tt, err)
				assert.Equal(tt, tc.expect, alerts)
			}
		})
	}
}
