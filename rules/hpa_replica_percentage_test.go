package rules

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kouzoh/merlin/alert"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
	"github.com/kouzoh/merlin/mocks"
)

func Test_hpaReplicaPercentageRuleBasic(t *testing.T) {
	notification := merlinv1.Notification{
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
			ruleName:   "ClusterRuleHPAReplicaPercentage/test-r",
			rule: &hpaReplicaPercentageClusterRule{
				resource: &merlinv1.ClusterRuleHPAReplicaPercentage{
					ObjectMeta: metav1.ObjectMeta{Name: "test-r"},
					Spec: merlinv1.ClusterRuleHPAReplicaPercentageSpec{
						Notification: notification,
					},
				},
			},
		},
		{
			desc:       "namespaceRule",
			objectMeta: metav1.ObjectMeta{Name: "test-r"},
			ruleName:   "RuleHPAReplicaPercentage/test-r",
			rule: &hpaReplicaPercentageNamespaceRule{
				resource: &merlinv1.RuleHPAReplicaPercentage{
					ObjectMeta: metav1.ObjectMeta{Name: "test-r"},
					Spec: merlinv1.RuleHPAReplicaPercentageSpec{
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

func Test_HPAReplicaPercentageRule_NewRule(t *testing.T) {
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
			ruleFactory: &HPAReplicaPercentageRule{},
			mockCall: func(key client.ObjectKey) runtime.Object {
				merlinRule := merlinv1.ClusterRuleHPAReplicaPercentage{
					ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name},
					Spec: merlinv1.ClusterRuleHPAReplicaPercentageSpec{
						Notification: merlinv1.Notification{
							Notifiers:  []string{"testNotifier"},
							Suppressed: true,
						},
					},
				}
				mockClient.EXPECT().
					Get(ctx, key, &merlinv1.ClusterRuleHPAReplicaPercentage{}).
					SetArg(2, merlinRule).
					Return(nil).
					Times(1)
				return &merlinRule
			},
		},
		{
			desc:        "namespaceRule",
			key:         client.ObjectKey{Namespace: "test-ns", Name: "test-rule"},
			ruleFactory: &HPAReplicaPercentageRule{},
			mockCall: func(key client.ObjectKey) runtime.Object {
				merlinRule := merlinv1.RuleHPAReplicaPercentage{
					ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name},
					Spec: merlinv1.RuleHPAReplicaPercentageSpec{
						Notification: merlinv1.Notification{
							Notifiers:  []string{"testNotifier"},
							Suppressed: true,
						},
					},
				}
				mockClient.EXPECT().
					Get(ctx, key, &merlinv1.RuleHPAReplicaPercentage{}).
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
			delay, err := r.GetDelaySeconds(&autoscalingv1.HorizontalPodAutoscaler{})
			assert.NoError(tt, err)
			assert.Equal(tt, time.Duration(0), delay)
		})
	}
}

func Test_HPAReplicaPercentageRule_Evaluate(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	ruleFactory := &HPAReplicaPercentageRule{}
	notification := merlinv1.Notification{Notifiers: []string{"testNotifier"}}
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
			desc: "clusterRule - non hpa should return err",
			key:  clusterRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().Get(ctx, clusterRuleKey, &merlinv1.ClusterRuleHPAReplicaPercentage{}).Return(nil),
			},
			resource:  &autoscalingv1.HorizontalPodAutoscalerList{},
			expectErr: true,
		},
		{
			desc: "namespaceRule - non hpa should return err",
			key:  namespaceRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().Get(ctx, namespaceRuleKey, &merlinv1.RuleHPAReplicaPercentage{}).Return(nil),
			},
			resource:  &autoscalingv1.HorizontalPodAutoscalerList{},
			expectErr: true,
		},
		{
			desc: "clusterRule - non exceeds hpa should not return alert",
			key:  clusterRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, clusterRuleKey, &merlinv1.ClusterRuleHPAReplicaPercentage{}).
					SetArg(2, merlinv1.ClusterRuleHPAReplicaPercentage{
						Spec: merlinv1.ClusterRuleHPAReplicaPercentageSpec{
							Notification: notification,
							Percent:      80,
						},
					}).
					Return(nil),
			},
			resource: &autoscalingv1.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "testHPA", Namespace: clusterRuleKey.Namespace},
				Spec:       autoscalingv1.HorizontalPodAutoscalerSpec{MaxReplicas: 10},
				Status:     autoscalingv1.HorizontalPodAutoscalerStatus{CurrentReplicas: 7},
			},
			expect: alert.Alert{
				Message:      "HPA percentage is within threshold (< 80%)",
				ResourceKind: "HorizontalPodAutoscaler",
				ResourceName: clusterRuleKey.Namespace + "/testHPA",
				Violated:     false,
			},
		},
		{
			desc: "clusterRule - exceeds hpa should not return alert",
			key:  clusterRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, clusterRuleKey, &merlinv1.ClusterRuleHPAReplicaPercentage{}).
					SetArg(2, merlinv1.ClusterRuleHPAReplicaPercentage{
						Spec: merlinv1.ClusterRuleHPAReplicaPercentageSpec{
							Notification: notification,
							Percent:      80,
						},
					}).
					Return(nil),
			},
			resource: &autoscalingv1.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "testHPA", Namespace: clusterRuleKey.Namespace},
				Spec:       autoscalingv1.HorizontalPodAutoscalerSpec{MaxReplicas: 10},
				Status:     autoscalingv1.HorizontalPodAutoscalerStatus{CurrentReplicas: 8},
			},
			expect: alert.Alert{
				Message:      "HPA percentage is >= 80%",
				ResourceKind: "HorizontalPodAutoscaler",
				ResourceName: clusterRuleKey.Namespace + "/testHPA",
				Violated:     true,
			},
		},
		{
			desc: "namespaceRule - non exceeds hpa should not return alert",
			key:  namespaceRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, namespaceRuleKey, &merlinv1.RuleHPAReplicaPercentage{}).
					SetArg(2, merlinv1.RuleHPAReplicaPercentage{
						Spec: merlinv1.RuleHPAReplicaPercentageSpec{
							Notification: notification,
							Percent:      80,
						},
					}).
					Return(nil),
			},
			resource: &autoscalingv1.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "testHPA", Namespace: namespaceRuleKey.Namespace},
				Spec:       autoscalingv1.HorizontalPodAutoscalerSpec{MaxReplicas: 10},
				Status:     autoscalingv1.HorizontalPodAutoscalerStatus{CurrentReplicas: 7},
			},
			expect: alert.Alert{
				Message:      "HPA percentage is within threshold (< 80%)",
				ResourceKind: "HorizontalPodAutoscaler",
				ResourceName: namespaceRuleKey.Namespace + "/testHPA",
				Violated:     false,
			},
		},
		{
			desc: "namespaceRule - exceeds hpa should not return alert",
			key:  namespaceRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, namespaceRuleKey, &merlinv1.RuleHPAReplicaPercentage{}).
					SetArg(2, merlinv1.RuleHPAReplicaPercentage{
						Spec: merlinv1.RuleHPAReplicaPercentageSpec{
							Notification: notification,
							Percent:      80,
						},
					}).
					Return(nil),
			},
			resource: &autoscalingv1.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "testHPA", Namespace: namespaceRuleKey.Namespace},
				Spec:       autoscalingv1.HorizontalPodAutoscalerSpec{MaxReplicas: 10},
				Status:     autoscalingv1.HorizontalPodAutoscalerStatus{CurrentReplicas: 8},
			},
			expect: alert.Alert{
				Message:      "HPA percentage is >= 80%",
				ResourceKind: "HorizontalPodAutoscaler",
				ResourceName: namespaceRuleKey.Namespace + "/testHPA",
				Violated:     true,
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

func Test_HPAReplicaPercentageRule_EvaluateAll(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	ruleFactory := &HPAReplicaPercentageRule{}
	notification := merlinv1.Notification{Notifiers: []string{"testNotifier"}}
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
			desc: "clusterRule - no hpa returns nil alerts",
			key:  clusterRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, clusterRuleKey, &merlinv1.ClusterRuleHPAReplicaPercentage{}).
					SetArg(2, merlinv1.ClusterRuleHPAReplicaPercentage{
						ObjectMeta: metav1.ObjectMeta{Namespace: clusterRuleKey.Namespace, Name: "cRule"},
						Spec: merlinv1.ClusterRuleHPAReplicaPercentageSpec{
							Notification: notification,
							Percent:      80,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &autoscalingv1.HorizontalPodAutoscalerList{}).
					Return(nil),
			},
		},
		{
			desc: "namespaceRule - no hpa returns nil alerts",
			key:  namespaceRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, namespaceRuleKey, &merlinv1.RuleHPAReplicaPercentage{}).
					SetArg(2, merlinv1.RuleHPAReplicaPercentage{
						ObjectMeta: metav1.ObjectMeta{Namespace: namespaceRuleKey.Namespace, Name: "nsRule"},
						Spec: merlinv1.RuleHPAReplicaPercentageSpec{
							Notification: notification,
							Percent:      80,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &autoscalingv1.HorizontalPodAutoscalerList{},
						&client.ListOptions{Namespace: namespaceRuleKey.Namespace}).
					Return(nil),
			},
		},
		{
			desc: "clusterRule - violated hpa should have alert violated to true",
			key:  clusterRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, clusterRuleKey, &merlinv1.ClusterRuleHPAReplicaPercentage{}).
					SetArg(2, merlinv1.ClusterRuleHPAReplicaPercentage{
						Spec: merlinv1.ClusterRuleHPAReplicaPercentageSpec{
							Notification: notification,
							Percent:      80,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &autoscalingv1.HorizontalPodAutoscalerList{}).
					SetArg(1, autoscalingv1.HorizontalPodAutoscalerList{
						Items: []autoscalingv1.HorizontalPodAutoscaler{
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hpa1"},
								Spec:       autoscalingv1.HorizontalPodAutoscalerSpec{MaxReplicas: 5},
								Status:     autoscalingv1.HorizontalPodAutoscalerStatus{CurrentReplicas: 3},
							},
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hpa2"},
								Spec:       autoscalingv1.HorizontalPodAutoscalerSpec{MaxReplicas: 5},
								Status:     autoscalingv1.HorizontalPodAutoscalerStatus{CurrentReplicas: 4},
							},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{Message: "HPA percentage is within threshold (< 80%)", ResourceKind: "HorizontalPodAutoscaler", ResourceName: "default/hpa1"},
				{Message: "HPA percentage is >= 80%", ResourceKind: "HorizontalPodAutoscaler", ResourceName: "default/hpa2", Violated: true},
			},
		},
		{
			desc: "namespaceRule - violated hpa should have alert violated to true",
			key:  namespaceRuleKey,
			mockCalls: []*gomock.Call{
				mockClient.EXPECT().
					Get(ctx, namespaceRuleKey, &merlinv1.RuleHPAReplicaPercentage{}).
					SetArg(2, merlinv1.RuleHPAReplicaPercentage{
						ObjectMeta: metav1.ObjectMeta{Namespace: namespaceRuleKey.Namespace, Name: "nsRule"},
						Spec: merlinv1.RuleHPAReplicaPercentageSpec{
							Notification: notification,
							Percent:      80,
						},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &autoscalingv1.HorizontalPodAutoscalerList{},
						&client.ListOptions{Namespace: namespaceRuleKey.Namespace}).
					SetArg(1, autoscalingv1.HorizontalPodAutoscalerList{
						Items: []autoscalingv1.HorizontalPodAutoscaler{
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: namespaceRuleKey.Namespace, Name: "hpa1"},
								Spec:       autoscalingv1.HorizontalPodAutoscalerSpec{MaxReplicas: 10},
								Status:     autoscalingv1.HorizontalPodAutoscalerStatus{CurrentReplicas: 0},
							},
							{
								ObjectMeta: metav1.ObjectMeta{Namespace: namespaceRuleKey.Namespace, Name: "hpa2"},
								Spec:       autoscalingv1.HorizontalPodAutoscalerSpec{MaxReplicas: 10},
								Status:     autoscalingv1.HorizontalPodAutoscalerStatus{CurrentReplicas: 8},
							},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{Message: "HPA percentage is within threshold (< 80%)", ResourceKind: "HorizontalPodAutoscaler", ResourceName: "testNS/hpa1"},
				{Message: "HPA percentage is >= 80%", ResourceKind: "HorizontalPodAutoscaler", ResourceName: "testNS/hpa2", Violated: true},
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
