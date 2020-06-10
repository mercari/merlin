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
)

func Test_Status(t *testing.T) {
	s := Status{}
	s.SetViolation(client.ObjectKey{Namespace: "ns", Name: "test"}, true)
	assert.Less(t, time.Now().Sub(*s.CheckedAt).Seconds(), float64(1))
	assert.Less(t, time.Now().Sub(s.Violations["ns/test"]).Seconds(), float64(1))
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
		selector    Selector
		namespace   string
		listOptions *client.ListOptions
	}{
		{
			desc:      "Selector with name should set fieldSelector as name in listOptions",
			selector:  Selector{Name: "test"},
			namespace: "default",
			listOptions: &client.ListOptions{
				Namespace:     "default",
				FieldSelector: fields.Set{".metadata.name": "test"}.AsSelector(),
			},
		},
		{
			desc:      "Selector with MatchLabels should set ",
			selector:  Selector{MatchLabels: map[string]string{"app": "test"}},
			namespace: "default",
			listOptions: &client.ListOptions{
				Namespace:     "default",
				LabelSelector: labels.SelectorFromSet(map[string]string{"app": "test"}),
			},
		},
		{
			desc:      "Selector should respect namespace value given",
			selector:  Selector{},
			namespace: "kube-system",
			listOptions: &client.ListOptions{
				Namespace: "kube-system",
			},
		},
		{
			desc:      "Selector Name and MatchLabels can co-exist",
			selector:  Selector{Name: "test", MatchLabels: map[string]string{"app": "test"}},
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
			assert.Equal(tt, tc.selector.AsListOption(tc.namespace), tc.listOptions)
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
