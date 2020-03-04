package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var notifierReconciler *NotifierReconciler

func SetupReconcilers(mgr manager.Manager) error {
	notifierReconciler = &NotifierReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("ctrl").WithName("Notifier"),
		Scheme: mgr.GetScheme(),
	}
	if err := notifierReconciler.SetupWithManager(mgr); err != nil {
		return err
	}
	if err := (&PodReconciler{
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("ctrl").WithName("Pod"),
		Scheme:    mgr.GetScheme(),
		Notifiers: notifierReconciler.NotifiersCache,
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	if err := (&HorizontalPodAutoscalerReconciler{
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("ctrl").WithName("HPA"),
		Scheme:    mgr.GetScheme(),
		Notifiers: notifierReconciler.NotifiersCache,
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	if err := (&NamespaceReconciler{
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("ctrl").WithName("Namespace"),
		Scheme:    mgr.GetScheme(),
		Notifiers: notifierReconciler.NotifiersCache,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
