package rules

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	merlinv1beta1 "github.com/kouzoh/merlin/api/v1beta1"
)

func Test_Status(t *testing.T) {
	s := Status{}
	s.setViolation(client.ObjectKey{Namespace: "ns", Name: "test"}, true)
	assert.Less(t, time.Now().Sub(*s.checkedAt).Seconds(), float64(1))
	assert.Less(t, time.Now().Sub(s.violations["ns/test"]).Seconds(), float64(1))
}

func Test_Rule(t *testing.T) {
	testEnv := &envtest.Environment{}
	cfg, err := testEnv.Start()
	assert.NoError(t, err)
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "127.0.0.1:8081",
		LeaderElection:     false,
		Port:               9444,
	})
	log := ctrl.Log.WithName("Rule")
	assert.NoError(t, err)
	r := &rule{cli: mgr.GetClient(), log: log, status: &Status{}}
	assert.NotNil(t, r.status)
	assert.False(t, r.IsReady())
	r.SetReady(true)
	assert.True(t, r.IsReady())
	r.SetReady(false)
	assert.False(t, r.IsReady())
}

func Test_Selector(t *testing.T) {
	cases := []struct {
		desc        string
		selector    merlinv1beta1.Selector
		namespace   string
		listOptions *client.ListOptions
	}{
		{
			desc:      "Selector with name should set fieldSelector as name in listOptions",
			selector:  merlinv1beta1.Selector{Name: "test"},
			namespace: "default",
			listOptions: &client.ListOptions{
				Namespace:     "default",
				FieldSelector: fields.Set{".metadata.name": "test"}.AsSelector(),
			},
		},
		{
			desc:      "Selector with MatchLabels should set ",
			selector:  merlinv1beta1.Selector{MatchLabels: map[string]string{"app": "test"}},
			namespace: "default",
			listOptions: &client.ListOptions{
				Namespace:     "default",
				LabelSelector: labels.SelectorFromSet(map[string]string{"app": "test"}),
			},
		},
		{
			desc:      "Selector should respect namespace value given",
			selector:  merlinv1beta1.Selector{},
			namespace: "kube-system",
			listOptions: &client.ListOptions{
				Namespace: "kube-system",
			},
		},
		{
			desc:      "Selector Name and MatchLabels can co-exist",
			selector:  merlinv1beta1.Selector{Name: "test", MatchLabels: map[string]string{"app": "test"}},
			namespace: "test",
			listOptions: &client.ListOptions{
				Namespace:     "test",
				FieldSelector: fields.Set{".metadata.name": "test"}.AsSelector(),
				LabelSelector: labels.SelectorFromSet(map[string]string{"app": "test"}),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			assert.Equal(tt, getListOptions(tc.selector, tc.namespace), tc.listOptions)
		})
	}
}

func Test_removeString(t *testing.T) {
	s := []string{"a", "b", "c", "d"}
	s = removeString(s, "b")
	assert.NotContains(t, s, "b")
	s = removeString(s, "a")
	assert.NotContains(t, s, "a")
	assert.Equal(t, s, []string{"c", "d"})
}

func Test_isStringInSlice(t *testing.T) {
	s := []string{"a", "b", "c"}
	assert.Equal(t, true, isStringInSlice(s, "a"))
	assert.Equal(t, true, isStringInSlice(s, "b"))
	assert.Equal(t, true, isStringInSlice(s, "c"))
	assert.Equal(t, false, isStringInSlice(s, "d"))
}

func Test_getStructName(t *testing.T) {
	type A struct{}
	cases := []struct {
		obj  interface{}
		name string
	}{
		{obj: &A{}, name: "A"},
		{obj: A{}, name: "A"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			assert.Equal(tt, tc.name, getStructName(tc.obj))
		})
	}
}

func Test_validateRequiredLabel(t *testing.T) {
	cases := []struct {
		desc           string
		requiredLabels merlinv1beta1.RequiredLabel
		labels         map[string]string
		message        string
		expectErr      bool
	}{
		{
			desc:           "empty labels should get proper message",
			requiredLabels: merlinv1beta1.RequiredLabel{Key: "test", Value: "test"},
			message:        "doenst have required label `test`",
		},
		{
			desc:           "non exists key should get proper message",
			requiredLabels: merlinv1beta1.RequiredLabel{Key: "test", Value: "test"},
			labels:         map[string]string{"blah": "test"},
			message:        "doenst have required label `test`",
		},
		{
			desc:           "incorrect value should get message without specified match",
			requiredLabels: merlinv1beta1.RequiredLabel{Key: "test", Value: "test"},
			labels:         map[string]string{"test": "blah"},
			message:        "has incorrect label value `blah` (expect `test`) for label `test`",
		},
		{
			desc:           "correct value should get message without specified match",
			requiredLabels: merlinv1beta1.RequiredLabel{Key: "test", Value: "test"},
			labels:         map[string]string{"test": "test"},
		},
		{
			desc:           "incorrect value should get message with match specified as exact",
			requiredLabels: merlinv1beta1.RequiredLabel{Key: "test", Value: "test", Match: "exact"},
			labels:         map[string]string{"test": "blah"},
			message:        "has incorrect label value `blah` (expect `test`) for label `test`",
		},
		{
			desc:           "correct value should get message with match specified as exact",
			requiredLabels: merlinv1beta1.RequiredLabel{Key: "test", Value: "test", Match: "exact"},
			labels:         map[string]string{"test": "test"},
		},
		{
			desc:           "incorrect value should get message with match specified as regexp",
			requiredLabels: merlinv1beta1.RequiredLabel{Key: "test", Value: "test", Match: "regexp"},
			labels:         map[string]string{"test": "blah"},
			message:        "has incorrect label value `blah` (regex match `test`) for label `test`",
		},
		{
			desc:           "correct value should get message with match specified as regexp",
			requiredLabels: merlinv1beta1.RequiredLabel{Key: "test", Value: "test", Match: "regexp"},
			labels:         map[string]string{"test": "test"},
		},
		{
			desc:           "correct value should get message with match specified as regexp",
			requiredLabels: merlinv1beta1.RequiredLabel{Key: "test", Value: "t[a-z]+", Match: "regexp"},
			labels:         map[string]string{"test": "test"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			result, err := validateRequiredLabel(tc.requiredLabels, tc.labels)
			if tc.expectErr {
				assert.Error(tt, err)
			} else {
				assert.NoError(tt, err)
				assert.Equal(tt, tc.message, result)
			}

		})
	}
}
