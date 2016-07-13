package controller

import (
	"github.com/golang/glog"

	"sitepod.io/sitepod/pkg/api/v1"
	"sitepod.io/sitepod/pkg/controller/etc"
	"sitepod.io/sitepod/pkg/controller/services"
	"sitepod.io/sitepod/pkg/controller/sitepod"
	"sitepod.io/sitepod/pkg/controller/systemuser"

	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/controller/framework"
)

type SingleNodeController struct {
	config             *SingleNodeConfig
	sitepodInformer    framework.SharedIndexInformer
	podInformer        framework.SharedIndexInformer
	pvInformer         framework.SharedIndexInformer
	systemUserInformer framework.SharedIndexInformer
	servicesInformer   framework.SharedIndexInformer
	configMapInformer  framework.SharedIndexInformer
	deploymentInformer framework.SharedIndexInformer

	homedirController    framework.ControllerInterface
	sitepodController    framework.ControllerInterface
	servicesController   framework.ControllerInterface
	etcController        framework.ControllerInterface
	systemUserController framework.ControllerInterface

	coreConcepts *v1.CoreConcepts
	extConcepts  *v1.ExtConcepts
	concepts     *v1.Concepts
}

func NewSingleNodeController(config *SingleNodeConfig) *SingleNodeController {

	controller := &SingleNodeController{config: config}
	controller.sitepodInformer = config.SitepodInformer
	controller.podInformer = config.PodInformer
	controller.systemUserInformer = config.SystemUserInformer
	controller.servicesInformer = config.ServicesInformer
	controller.configMapInformer = config.ConfigMapInformer
	controller.deploymentInformer = config.DeploymentInformer
	controller.pvInformer = config.PvInformer
	controller.concepts = config.Concepts
	controller.coreConcepts = config.CoreConcepts
	controller.extConcepts = config.ExtConcepts

	return controller
}

func (c *SingleNodeController) Run(stopCh <-chan struct{}) {
	glog.Infof("Starting single node controller")
	glog.Infof("Verifying single node cluster exists")

	_, err := c.concepts.Clusters.Getter("singlenode")
	if err != nil {

		if errors.IsNotFound(err) {
			glog.Infof("Creating initial singlenode cluster resource")
			//TODO verify error is actually a 404
			// Create the initial cluster if not existing
			newCluster := &v1.Cluster{
				Spec: v1.ClusterSpec{
					DisplayName:  "Single Node Cluster",
					Description:  "Single Node Cluster",
					FileUIDCount: 2001,
				}}
			newCluster.Name = "singlenode"

			_, err = c.concepts.Clusters.Adder(newCluster)
			if err != nil {
				panic("Unable to register cluster, did you create the TPRS?")
			}
		} else {
			glog.Errorf("Unable to start: %+v", err)
			return
		}
	}

	c.etcController = etc.NewEtcController(
		c.sitepodInformer,
		c.systemUserInformer,
		c.coreConcepts.ConfigMaps.Getter,
		c.coreConcepts.ConfigMaps.Updater)
	go c.etcController.Run(stopCh)

	c.servicesController = services.NewServicesController(
		c.servicesInformer,
		c.sitepodInformer,
		c.deploymentInformer,
		c.pvInformer,
		c.configMapInformer,
		c.extConcepts.Deployments.Updater,
		c.coreConcepts.ConfigMaps.Updater,
	)
	go c.servicesController.Run(stopCh)

	c.sitepodController = sitepod.NewSitepodController(
		c.sitepodInformer,
		c.pvInformer,
		c.deploymentInformer,
		c.concepts.Sitepods.Updater,
		c.coreConcepts.Rcs.Updater,
		c.coreConcepts.PersistentVolumes.Updater,
		c.extConcepts.Deployments.Updater,
		c.extConcepts.Deployments.Deleter,
		c.extConcepts.ReplicaSets.GetWithLabels,
		c.extConcepts.ReplicaSets.Deleter,
	)
	go c.sitepodController.Run(stopCh)

	c.systemUserController = systemuser.NewSystemUserController(c.sitepodInformer,
		c.systemUserInformer,
		c.pvInformer,
		c.concepts.Clusters.Getter,
		c.concepts.Clusters.Updater,
		c.concepts.SystemUsers.Updater,
	)

	go c.systemUserController.Run(stopCh)

	go c.systemUserInformer.Run(stopCh)
	go c.servicesInformer.Run(stopCh)
	go c.pvInformer.Run(stopCh)
	go c.podInformer.Run(stopCh)
	go c.deploymentInformer.Run(stopCh)
	go c.sitepodInformer.Run(stopCh)
	go c.configMapInformer.Run(stopCh)

	glog.Info("Waiting to stop")
	<-stopCh
	glog.Info("Stopped")
}

func (c *SingleNodeController) HasSynced() bool {
	return true
}
