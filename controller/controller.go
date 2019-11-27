package controller

import (
	"context"
	"go.uber.org/zap"
	"k8s.io/api/apps/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

type Controller struct {
	Clientset *kubernetes.Clientset
	Interval  time.Duration
	Logger    *zap.Logger
}

type PodInfo struct {
	Name       string
	Deployment string
	ReplicaSet string
	Service    string
	HPA        string
	PDB        string
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
		c.Logger.Info("checking namespace", zap.String("ns", ns.Name))
		var podInfos []PodInfo

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

		//// HPAs
		//hpas, err := c.Clientset.AutoscalingV1().HorizontalPodAutoscalers(ns.Name).List(metav1.ListOptions{})
		//if err != nil {
		//	c.Logger.Error("failed to list HPAs", zap.Error(err), zap.String("ns", ns.Name))
		//}

		for _, p := range pods.Items {
			podInfo := PodInfo{Name: p.Name}

			// checking if pod has too many restarts and not running
			for _, containerStatus := range p.Status.ContainerStatuses {
				if containerStatus.RestartCount > 10 && p.Status.Phase != v1.PodRunning {
					c.Logger.Warn("Pod has >10 restarts and it's not running",
						zap.String("ns", ns.Name),
						zap.String("pod", p.Name))
				}
			}

			// checking what deployment the pod belongs to
			for _, d := range deployments.Items {
				matches := 0
				for k, v := range d.Spec.Selector.MatchLabels {
					if _, ok := p.GetObjectMeta().GetLabels()[k]; ok && v == p.GetObjectMeta().GetLabels()[k] {
						matches += 1
					}
				}
				if matches == len(d.Spec.Selector.MatchLabels) {
					podInfo.Deployment = d.Name
				}
			}

			// checking what replicaset the pod belongs to
			for _, r := range replicaSets.Items {
				matches := 0
				for k, v := range r.Spec.Selector.MatchLabels {
					if _, ok := p.GetObjectMeta().GetLabels()[k]; ok && v == p.GetObjectMeta().GetLabels()[k] {
						matches += 1
					}
				}
				if matches == len(r.Spec.Selector.MatchLabels) {
					podInfo.ReplicaSet = r.Name
				}
			}

			if podInfo.Deployment == "" && podInfo.ReplicaSet == "" {
				c.Logger.Warn("Pod is not managed by a deployment or replicaset",
					zap.String("ns", ns.Name),
					zap.String("pod", p.Name))
			}

			// checking what service the pod belongs to
			for _, s := range services.Items {
				matches := 0
				for k, v := range s.Spec.Selector {
					if _, ok := p.GetObjectMeta().GetLabels()[k]; ok && v == p.GetObjectMeta().GetLabels()[k] {
						matches += 1
					}
				}
				if matches == len(s.Spec.Selector) {
					podInfo.Service = s.Name
				}
			}

			if podInfo.Service == "" {
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

			// checking if what pdb the pod belongs to
			for _, pdb := range pdbs.Items {
				matches := 0
				for k, v := range pdb.Spec.Selector.MatchLabels {
					if _, ok := p.GetObjectMeta().GetLabels()[k]; ok && v == p.GetObjectMeta().GetLabels()[k] {
						matches += 1
					}
				}
				if matches == len(pdb.Spec.Selector.MatchLabels) {
					podInfo.PDB = pdb.Name
				}
			}
			if podInfo.PDB == "" {
				c.Logger.Warn("Pod is not managed by PDB",
					zap.String("ns", ns.Name),
					zap.String("pod", p.Name))
			}

			podInfos = append(podInfos, podInfo)
		}

		// Check orphaned resources, like service, hpa, pdb, etc
		//for _, d := range deployments.Items {
		//	c.Logger.Debug("checking deployment", zap.String("ns", ns.Name), zap.String("deploy", d.Name))
		//}
		//
		//
		//for _, r := range replicaSets.Items {
		//	c.Logger.Debug("checking replicaset", zap.String("ns", ns.Name), zap.String("replicaset", r.Name))
		//}
		//
		//
		//for _, s := range services.Items {
		//	c.Logger.Debug("checking service", zap.String("ns", ns.Name), zap.String("service", s.Name))
		//}
		//
		//
		//for _, p := range pdbs.Items {
		//	c.Logger.Debug("checking PDB", zap.String("ns", ns.Name), zap.String("pdb", p.Name))
		//}
		//
		//
		//for _, h := range hpas.Items {
		//	c.Logger.Debug("checking PDB", zap.String("ns", ns.Name), zap.String("hpa", h.Name))
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
