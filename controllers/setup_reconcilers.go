package controllers

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sync"
	"time"

	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// Notifiers stores the notifiers as cache, this will be updated when any notifier updates happen,
	// and also servers as cache so we dont need to get list of notifiers every time
	Notifiers map[string]*merlinv1.Notifier
	// Generations stores the rule generation, to be used for event filter to determine if events are from Reconciler
	// This is required since status updates also increases generation, so we cant use metadata's generation.
	Generations *sync.Map
	// RuleStatues stores the status of rules, it has sync.Mutex so reconciler process needs to acquire the lock
	// before making changes
	RuleStatues map[string]*RuleStatusWithLock
}

var notifierReconciler *NotifierReconciler

func SetupReconcilers(mgr manager.Manager) error {
	notifierReconciler = &NotifierReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("ctrl").WithName("Notifier"),
		Scheme:     mgr.GetScheme(),
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
	if err := notifierReconciler.SetupWithManager(mgr); err != nil {
		return err
	}
	if err := (&PodReconciler{
		Reconciler{
			Client:    mgr.GetClient(),
			Log:       ctrl.Log.WithName("ctrl").WithName("Pod"),
			Scheme:    mgr.GetScheme(),
			Notifiers: notifierReconciler.Notifiers,
		},
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	if err := (&HorizontalPodAutoscalerReconciler{
		Reconciler{
			Client:    mgr.GetClient(),
			Log:       ctrl.Log.WithName("ctrl").WithName("HPA"),
			Scheme:    mgr.GetScheme(),
			Notifiers: notifierReconciler.Notifiers,
		},
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	if err := (&NamespaceReconciler{
		Reconciler{
			Client:    mgr.GetClient(),
			Log:       ctrl.Log.WithName("ctrl").WithName("Namespace"),
			Scheme:    mgr.GetScheme(),
			Notifiers: notifierReconciler.Notifiers,
		},
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	if err := (&PodDisruptionBudgetReconciler{
		Reconciler{Client: mgr.GetClient(),
			Log:       ctrl.Log.WithName("ctrl").WithName("PDB"),
			Scheme:    mgr.GetScheme(),
			Notifiers: notifierReconciler.Notifiers,
		},
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&ServiceReconciler{
		Reconciler{Client: mgr.GetClient(),
			Log:       ctrl.Log.WithName("ctrl").WithName("Service"),
			Scheme:    mgr.GetScheme(),
			Notifiers: notifierReconciler.Notifiers,
		},
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
