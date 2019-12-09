package controller

import (
	"context"
	"go.uber.org/zap"
	"k8s.io/api/apps/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

const (
	IstioInjectionLabel = "istio-injection"
)

var SkipNamespaces = []string{
	"cert-manager",
	"certificate-expiry-monitor-controller",
	"istio-system",
	"kube-system",
	"prometheus",
}

type Controller struct {
	Clientset *kubernetes.Clientset
	Interval  time.Duration
	Logger    *zap.Logger
}

type ServiceInfo struct {
	Name       string
	NameSpace  string
	Deployment string
	ReplicaSet string
	Service    string
	HPA        string
	PDB        string
	NumPods    int32
}

func (c *Controller) GetBelongedDeployment(pods, deployment *v1beta1.Deployment) (interface{}, error) {
	return nil, nil
}

// RunOnce runs a single iteration of a reconciliation loop.
func (c *Controller) RunOnce(ctx context.Context) error {
	namespaces, err := c.Clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		c.Logger.Error("failed to list namespaces", zap.Error(err))
	}

	for _, ns := range namespaces.Items {
		if IsSkipCheckingNamespace(ns.Name) {
			c.Logger.Debug("Skip checking namespace as it's in the skip list.",
				zap.String("ns", ns.Name))
			continue
		}
		ServiceInfos := map[string]*ServiceInfo{}
		c.Logger.Info("check namespace", zap.String("ns", ns.Name))

		// check if ns has istio-inject
		if ns.ObjectMeta.Labels[IstioInjectionLabel] == "" {
			c.Logger.Warn("Namespaces has no istio injection and it's not explicitly disabled",
				zap.String("ns", ns.Name))
		}

		// pods
		pods, err := c.Clientset.CoreV1().Pods(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			c.Logger.Error("failed to list pods", zap.Error(err), zap.String("ns", ns.Name))
		}

		// deployments
		deployments, err := c.Clientset.ExtensionsV1beta1().Deployments(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			c.Logger.Error("failed to list deployments", zap.Error(err), zap.String("ns", ns.Name))
		}

		// replicasets
		replicaSets, err := c.Clientset.ExtensionsV1beta1().ReplicaSets(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			c.Logger.Error("failed to list replicaSets", zap.Error(err), zap.String("ns", ns.Name))
		}

		// services
		services, err := c.Clientset.CoreV1().Services(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			c.Logger.Error("failed to list services", zap.Error(err), zap.String("ns", ns.Name))
		}

		// PDBs
		pdbs, err := c.Clientset.PolicyV1beta1().PodDisruptionBudgets(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			c.Logger.Error("failed to list PDBs", zap.Error(err), zap.String("ns", ns.Name))
		}

		// HPAs
		hpas, err := c.Clientset.AutoscalingV1().HorizontalPodAutoscalers(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			c.Logger.Error("failed to list HPAs", zap.Error(err), zap.String("ns", ns.Name))
		}

		for _, p := range pods.Items {

			// check if pod has too many restarts and not running
			for _, containerStatus := range p.Status.ContainerStatuses {
				if containerStatus.RestartCount > 10 && p.Status.Phase != v1.PodRunning {
					c.Logger.Warn("Pod has >10 restarts and it's not running",
						zap.String("ns", ns.Name),
						zap.String("pod", p.Name))
				}
			}

			podNameSlice := strings.Split(p.Name, "-")
			podBaseName := strings.Join(podNameSlice[:len(podNameSlice)-1], "-")
			if s, ok := ServiceInfos[podBaseName]; ok {
				s.NumPods += 1
				// same type of pod already exists in the map
				continue
			}
			ServiceInfo := ServiceInfo{Name: podBaseName, NumPods: 1}

			// check what deployment the service pod belongs to
			for _, d := range deployments.Items {
				matches := 0
				for k, v := range d.Spec.Selector.MatchLabels {
					if _, ok := p.GetObjectMeta().GetLabels()[k]; ok && v == p.GetObjectMeta().GetLabels()[k] {
						matches += 1
					}
				}
				if matches == len(d.Spec.Selector.MatchLabels) {
					ServiceInfo.Deployment = d.Name
				}
			}

			// check what replicaset the pod belongs to
			for _, r := range replicaSets.Items {
				matches := 0
				for k, v := range r.Spec.Selector.MatchLabels {
					if _, ok := p.GetObjectMeta().GetLabels()[k]; ok && v == p.GetObjectMeta().GetLabels()[k] {
						matches += 1
					}
				}
				if matches == len(r.Spec.Selector.MatchLabels) {
					ServiceInfo.ReplicaSet = r.Name
				}
			}

			if ServiceInfo.Deployment == "" && ServiceInfo.ReplicaSet == "" {
				c.Logger.Warn("Pod is not managed by a deployment or replicaset",
					zap.String("ns", ns.Name),
					zap.String("pod", p.Name))
			}

			// check what service the pod belongs to
			for _, s := range services.Items {
				matches := 0
				for k, v := range s.Spec.Selector {
					if _, ok := p.GetObjectMeta().GetLabels()[k]; ok && v == p.GetObjectMeta().GetLabels()[k] {
						matches += 1
					}
				}
				if matches == len(s.Spec.Selector) {
					ServiceInfo.Service = s.Name
				}
			}

			if ServiceInfo.Service == "" {
				isJob := false
				for _, o := range p.OwnerReferences {
					if o.Kind == "Job" {
						isJob = true
					}
				}
				if !isJob {
					c.Logger.Warn("Pod is not used by a service",
						zap.String("ns", ns.Name),
						zap.String("pod", p.Name))
				}
			}

			// check what pdb the pod belongs to
			for _, pdb := range pdbs.Items {
				matches := 0
				for k, v := range pdb.Spec.Selector.MatchLabels {
					if _, ok := p.GetObjectMeta().GetLabels()[k]; ok && v == p.GetObjectMeta().GetLabels()[k] {
						matches += 1
					}
				}
				if matches == len(pdb.Spec.Selector.MatchLabels) {
					ServiceInfo.PDB = pdb.Name
				}
			}
			if ServiceInfo.PDB == "" {
				c.Logger.Warn("Pod is not managed by PDB",
					zap.String("ns", ns.Name),
					zap.String("pod", p.Name))
			}

			// check if the pod's replicaset or deployment has hpa
			for _, hpa := range hpas.Items {
				if hpa.Spec.ScaleTargetRef.Kind == "Deployment" {
					if ServiceInfo.Deployment == hpa.Spec.ScaleTargetRef.Name {
						ServiceInfo.HPA = hpa.Name
					}
				} else if hpa.Spec.ScaleTargetRef.Kind == "ReplicaSet" {
					if ServiceInfo.ReplicaSet == hpa.Spec.ScaleTargetRef.Name {
						ServiceInfo.HPA = hpa.Name
					}
				}
			}
			if ServiceInfo.HPA == "" {
				c.Logger.Warn("Pod is not managed by HPA",
					zap.String("ns", ns.Name),
					zap.String("pod", p.Name))
			}

			ServiceInfos[podBaseName] = &ServiceInfo
		}

		// Check orphaned resources, like service, hpa, pdb, etc
		//for _, d := range deployments.Items {
		//	c.Logger.Debug("check deployment", zap.String("ns", ns.Name), zap.String("deploy", d.Name))
		//}
		//
		//
		//for _, r := range replicaSets.Items {
		//	c.Logger.Debug("check replicaset", zap.String("ns", ns.Name), zap.String("replicaset", r.Name))
		//}
		//
		//
		//for _, s := range services.Items {
		//	c.Logger.Debug("check service", zap.String("ns", ns.Name), zap.String("service", s.Name))
		//}
		//
		//
		//for _, p := range pdbs.Items {
		//	c.Logger.Debug("check PDB", zap.String("ns", ns.Name), zap.String("pdb", p.Name))
		//}
		//
		//
		//for _, h := range hpas.Items {
		//	c.Logger.Debug("check PDB", zap.String("ns", ns.Name), zap.String("hpa", h.Name))
		//}
		//

	}
	return nil
}

// Run runs RunOnce in a loop with a delay until context is canceled
func (c *Controller) Run(ctx context.Context) {
	for {
		err := c.RunOnce(ctx)
		if err != nil {
			c.Logger.Error("failed to run", zap.Error(err))
		}

		select {
		case <-time.After(c.Interval):
		case <-ctx.Done():
			c.Logger.Info("terminating main controller loop")
			return
		}
	}
}

func IsSkipCheckingNamespace(namespace string) bool {
	for _, skipNamespace := range SkipNamespaces {
		if namespace == skipNamespace {
			return true
		}
	}
	return false
}
