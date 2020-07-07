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

func Test_SecretUnusedRuleBasic(t *testing.T) {
	notification := merlinv1beta1.Notification{
		Notifiers:  []string{"testNotifier"},
		Suppressed: true,
	}

	merlinv1beta1Rule := &merlinv1beta1.ClusterRuleSecretUnused{
		ObjectMeta: metav1.ObjectMeta{Name: "test-r"},
		Spec: merlinv1beta1.ClusterRuleSecretUnusedSpec{
			Notification: notification,
		},
	}

	r := &SecretUnusedRule{resource: merlinv1beta1Rule}
	assert.Equal(t, merlinv1beta1Rule.ObjectMeta, r.GetObjectMeta())
	assert.Equal(t, notification, r.GetNotification())
	assert.Equal(t, "ClusterRuleSecretUnused/test-r", r.GetName())

	finalizer := "test.finalizer"
	r.SetFinalizer(finalizer)
	assert.Equal(t, finalizer, r.resource.Finalizers[0])
	r.RemoveFinalizer(finalizer)
	assert.Empty(t, r.resource.Finalizers)
}

func Test_SecretUnusedRule_NewRule(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	key := client.ObjectKey{Namespace: "test-ns", Name: "test-rule"}

	merlinv1beta1Rule := merlinv1beta1.ClusterRuleSecretUnused{
		ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name},
		Spec: merlinv1beta1.ClusterRuleSecretUnusedSpec{
			Notification: merlinv1beta1.Notification{
				Notifiers:  []string{"testNotifier"},
				Suppressed: true,
			},
			InitialDelaySeconds: 30,
		},
	}
	rule := &SecretUnusedRule{}
	mockClient.EXPECT().Get(ctx, key, &merlinv1beta1.ClusterRuleSecretUnused{}).SetArg(2, merlinv1beta1Rule).Return(nil)
	r, err := rule.New(ctx, mockClient, log, key)
	assert.NoError(t, err)
	assert.Equal(t, &merlinv1beta1Rule, r.GetObject())
	delay, err := r.GetDelaySeconds(&corev1.Secret{
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

func Test_SecretUnusedRule_EvaluateAll(t *testing.T) {
	ctx := context.Background()
	log := zapr.NewLogger(zap.L())
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)

	r := &SecretUnusedRule{
		rule: rule{cli: mockClient, log: log, status: &Status{}},
		resource: &merlinv1beta1.ClusterRuleSecretUnused{
			ObjectMeta: metav1.ObjectMeta{Name: "test-r"},
			Spec: merlinv1beta1.ClusterRuleSecretUnusedSpec{
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
			desc: "Empty Secrets should return empty alerts",
			mock: []*gomock.Call{
				mockClient.EXPECT().List(ctx, &corev1.SecretList{}).Return(nil),
			},
			expect: nil,
		},
		{
			desc: "Unused Secrets in ignored namespace should not return an alert",
			mock: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.SecretList{}).
					SetArg(1, corev1.SecretList{
						Items: []corev1.Secret{{
							Type: corev1.SecretTypeOpaque,
							ObjectMeta: metav1.ObjectMeta{
								Namespace: r.resource.Spec.IgnoreNamespaces[0],
								Name:      "test-secret",
							},
						}},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Suppressed:   true,
					Message:      "namespace is ignored by the rule",
					ResourceKind: "Secret",
					ResourceName: "ignored-ns/test-secret",
					Violated:     false,
				},
			},
		},
		{
			desc: "Unused Secrets should return an alert",
			mock: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.SecretList{}).
					SetArg(1, corev1.SecretList{
						Items: []corev1.Secret{{
							Type: corev1.SecretTypeOpaque,
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "test-ns",
								Name:      "test-secret",
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
					Message:      "secret is not being used",
					ResourceKind: "Secret",
					ResourceName: "test-ns/test-secret",
					Violated:     true,
				},
			},
		},
		{
			desc: "Secret used in volume should return a non-violated alert",
			mock: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.SecretList{}).
					SetArg(1, corev1.SecretList{
						Items: []corev1.Secret{{
							Type: corev1.SecretTypeOpaque,
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "test-ns",
								Name:      "secret-in-vol",
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
									Name: "test-secret-in-vol",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{SecretName: "secret-in-vol"},
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
					Message:      "Secret is being used by pod 'alpine' volume 'test-secret-in-vol'",
					ResourceKind: "Secret",
					ResourceName: "test-ns/secret-in-vol",
					Violated:     false,
				},
			},
		},
		{
			desc: "Secret used in env-var should return a non-violated alert",
			mock: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.SecretList{}).
					SetArg(1, corev1.SecretList{
						Items: []corev1.Secret{{
							Type: corev1.SecretTypeOpaque,
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "test-ns",
								Name:      "secret-in-env",
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
										Name: "test-secret-env",
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "secret-in-env"}}},
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
					Message:      "Secret is being used by pod 'alpine' container 'b' env 'test-secret-env'",
					ResourceKind: "Secret",
					ResourceName: "test-ns/secret-in-env",
					Violated:     false,
				},
			},
		},
		{
			desc: "Secret used in env-source should return a non-violated alert",
			mock: []*gomock.Call{
				mockClient.EXPECT().
					List(ctx, &corev1.SecretList{}).
					SetArg(1, corev1.SecretList{
						Items: []corev1.Secret{{
							Type: corev1.SecretTypeOpaque,
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "test-ns",
								Name:      "secret-in-env-source",
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
										SecretRef: &corev1.SecretEnvSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "secret-in-env-source"}}}},
								}},
							}},
						},
					}).
					Return(nil),
			},
			expect: []alert.Alert{
				{
					Suppressed:   true,
					Message:      "Secret is being used by pod 'alpine' container 'c' env",
					ResourceKind: "Secret",
					ResourceName: "test-ns/secret-in-env-source",
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
