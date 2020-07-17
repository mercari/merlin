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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kouzoh/merlin/alert"
	merlinv1beta1 "github.com/kouzoh/merlin/api/v1beta1"
	"github.com/kouzoh/merlin/mocks"
)

func Test_ConfigMapUnusedRuleBasic(t *testing.T) {
	notification := merlinv1beta1.Notification{
		Notifiers:  []string{"testNotifier"},
		Suppressed: true,
	}

	merlinv1beta1Rule := &merlinv1beta1.ClusterRuleConfigMapUnused{
		ObjectMeta: metav1.ObjectMeta{Name: "test-r"},
		Spec: merlinv1beta1.ClusterRuleConfigMapUnusedSpec{
			Notification: notification,
		},
	}

	r := &ConfigMapUnusedRule{resource: merlinv1beta1Rule}
	assert.Equal(t, merlinv1beta1Rule.ObjectMeta, r.GetObjectMeta())
	assert.Equal(t, notification, r.GetNotification())
	assert.Equal(t, "ClusterRuleConfigMapUnused/test-r", r.GetName())

	finalizer := "test.finalizer"
	r.SetFinalizer(finalizer)
	assert.Equal(t, finalizer, r.resource.Finalizers[0])
	r.RemoveFinalizer(finalizer)
	assert.Empty(t, r.resource.Finalizers)
}

func Test_ConfigMapUnusedRule_NewRule(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	key := client.ObjectKey{Namespace: "test-ns", Name: "test-rule"}

	merlinv1beta1Rule := merlinv1beta1.ClusterRuleConfigMapUnused{
		ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name},
		Spec: merlinv1beta1.ClusterRuleConfigMapUnusedSpec{
			Notification: merlinv1beta1.Notification{
				Notifiers:  []string{"testNotifier"},
				Suppressed: true,
			},
			InitialDelaySeconds: 30,
		},
	}
	rule := &ConfigMapUnusedRule{}
	mockClient.EXPECT().Get(ctx, key, &merlinv1beta1.ClusterRuleConfigMapUnused{}).SetArg(2, merlinv1beta1Rule).Return(nil)
	r, err := rule.New(ctx, mockClient, log, key)
	assert.NoError(t, err)
	assert.Equal(t, &merlinv1beta1Rule, r.GetObject())
	delay, err := r.GetDelaySeconds(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(30*time.Second), delay)
	delay, err = r.GetDelaySeconds(&corev1.Pod{})
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), delay)
	_, err = r.GetDelaySeconds(&corev1.Namespace{})
	assert.Error(t, err)
}

func Test_ConfigMapUnusedRule_EvaluateAll(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)

	r := &ConfigMapUnusedRule{
		rule: rule{cli: mockClient, log: log, status: &Status{}},
		resource: &merlinv1beta1.ClusterRuleConfigMapUnused{
			ObjectMeta: metav1.ObjectMeta{Name: "test-r"},
			Spec: merlinv1beta1.ClusterRuleConfigMapUnusedSpec{
				IgnoreNamespaces: []string{"ignored-ns"},
				Notification: merlinv1beta1.Notification{
					Notifiers:  []string{"testNotifier"},
					Suppressed: true,
				},
			},
		},
	}

	cases := []struct {
		desc      string
		mock      []*gomock.Call
		expect    []alert.Alert
		expectErr bool
	}{
		{
			desc: "Empty ConfigMaps should return empty alerts",
			mock: []*gomock.Call{
				mockClient.EXPECT().List(ctx, &corev1.ConfigMapList{}).Return(nil),
			},
			expect: nil,
		},
		{
			desc: "Unused ConfigMaps in ignored namespace should not return an alert",
			mock: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.ConfigMapList{}).
					SetArg(1, corev1.ConfigMapList{
						Items: []corev1.ConfigMap{{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: r.resource.Spec.IgnoreNamespaces[0],
								Name:      "test-configMap",
							},
						}},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Suppressed:   true,
					Message:      "namespace is ignored by the rule",
					ResourceKind: "ConfigMap",
					ResourceName: "ignored-ns/test-configMap",
					Violated:     false,
				},
			},
		},
		{
			desc: "Unused ConfigMaps should return an alert",
			mock: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.ConfigMapList{}).
					SetArg(1, corev1.ConfigMapList{
						Items: []corev1.ConfigMap{{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "test-ns",
								Name:      "test-configMap",
							},
						}},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{Namespace: "test-ns"}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Suppressed:   true,
					Message:      "configMap is not being used",
					ResourceKind: "ConfigMap",
					ResourceName: "test-ns/test-configMap",
					Violated:     true,
				},
			},
		},
		{
			desc: "ConfigMap used in volume should return a non-violated alert",
			mock: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.ConfigMapList{}).
					SetArg(1, corev1.ConfigMapList{
						Items: []corev1.ConfigMap{{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "test-ns",
								Name:      "configMap-in-vol",
							},
						}},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{Namespace: "test-ns"}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "alpine",
								Namespace: "test-ns",
								Labels:    map[string]string{"app": "alpine"}},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{{
									Name: "test-configMap-in-vol",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "configMap-in-vol",
											},
										},
									},
								}},
								Containers: []corev1.Container{{
									Name:    "a",
									Image:   "alpine",
									Command: []string{"top"},
								}},
							}},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Suppressed:   true,
					Message:      "ConfigMap is being used by pod 'alpine' volume 'test-configMap-in-vol'",
					ResourceKind: "ConfigMap",
					ResourceName: "test-ns/configMap-in-vol",
					Violated:     false,
				},
			},
		},
		{
			desc: "ConfigMap used in env-var should return a non-violated alert",
			mock: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.ConfigMapList{}).
					SetArg(1, corev1.ConfigMapList{
						Items: []corev1.ConfigMap{{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "test-ns",
								Name:      "configMap-in-env",
							},
						}},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{Namespace: "test-ns"}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "alpine",
								Namespace: "test-ns",
								Labels:    map[string]string{"app": "alpine"}},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name:    "b",
									Image:   "alpine",
									Command: []string{"top"},
									Env: []corev1.EnvVar{{
										Name: "test-configMap-env",
										ValueFrom: &corev1.EnvVarSource{
											ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "configMap-in-env"}}},
									}},
								}},
							}},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Suppressed:   true,
					Message:      "ConfigMap is being used by pod 'alpine' container 'b' env 'test-configMap-env'",
					ResourceKind: "ConfigMap",
					ResourceName: "test-ns/configMap-in-env",
					Violated:     false,
				},
			},
		},
		{
			desc: "ConfigMap used in env-source should return a non-violated alert",
			mock: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.ConfigMapList{}).
					SetArg(1, corev1.ConfigMapList{
						Items: []corev1.ConfigMap{{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "test-ns",
								Name:      "configMap-in-env-source",
							},
						}},
					}).
					Return(nil),
				mockClient.EXPECT().
					List(ctx, &corev1.PodList{}, &client.ListOptions{Namespace: "test-ns"}).
					SetArg(1, corev1.PodList{
						Items: []corev1.Pod{{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "alpine",
								Namespace: "test-ns",
								Labels:    map[string]string{"app": "alpine"}},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name:    "c",
									Image:   "alpine",
									Command: []string{"top"},
									EnvFrom: []corev1.EnvFromSource{{
										ConfigMapRef: &corev1.ConfigMapEnvSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "configMap-in-env-source"}}}},
								}},
							}},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Suppressed:   true,
					Message:      "ConfigMap is being used by pod 'alpine' container 'c' env",
					ResourceKind: "ConfigMap",
					ResourceName: "test-ns/configMap-in-env-source",
					Violated:     false,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			for _, call := range tc.mock {
				call.Times(1)
			}
			alerts, err := r.EvaluateAll(ctx)
			if tc.expectErr {
				assert.Error(tt, err)
			} else {
				assert.NoError(tt, err)
			}
			assert.Equal(tt, tc.expect, alerts)
		})
	}
}
