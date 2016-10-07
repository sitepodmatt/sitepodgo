package system

import (
	"github.com/golang/glog"

	"sitepod.io/sitepod/pkg/client"
	//"sitepod.io/sitepod/pkg/controller/appcomp"
	//"sitepod.io/sitepod/pkg/controller/etc"
	//"sitepod.io/sitepod/pkg/controller/podtask"
	//"sitepod.io/sitepod/pkg/controller/sitepod"
	//"sitepod.io/sitepod/pkg/controller/systemuser"
	//"sitepod.io/sitepod/pkg/controller/website"

	"k8s.io/kubernetes/pkg/api"
	k8s_v1 "k8s.io/kubernetes/pkg/api/v1"
	ext_api "k8s.io/kubernetes/pkg/apis/extensions"
	ext_v1 "k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	"k8s.io/kubernetes/pkg/runtime"
	"sitepod.io/sitepod/pkg/api/v1"
	"sitepod.io/sitepod/pkg/webapi"
)

type SimpleSystem struct {
	Config *SimpleConfig
}

type SimpleConfig struct {
	ApiServer string
	Namespace string
}

func NewSimpleSystem(config *SimpleConfig) *SimpleSystem {
	return &SimpleSystem{config}
}

func BundleScheme() *runtime.Scheme {

	// TODO this is hacky, create a fresh scheme
	v1.AddToScheme(api.Scheme)
	k8s_v1.AddToScheme(api.Scheme)
	//api.AddToScheme(api.Scheme)
	ext_v1.AddToScheme(api.Scheme)
	ext_api.AddToScheme(api.Scheme)
	return api.Scheme
}

func (s *SimpleSystem) GetClient() *client.Client {

	cc := client.NewClient(BundleScheme(), &client.ClientConfig{s.Config.ApiServer,
		s.Config.Namespace})
	return cc

}

func (s *SimpleSystem) Run(stopCh <-chan struct{}) {
	glog.Info("Starting simple system")

	cc := s.GetClient()
	webInst := webapi.NewWebApi(cc)
	webInst.Start()

	//etcController := etc.NewEtcController(cc)
	//go etcController.Run(stopCh)

	//appCompController := appcomp.NewAppCompController(cc)
	//go appCompController.Run(stopCh)

	//sitepodController := sitepod.NewSitepodController(cc)
	//go sitepodController.Run(stopCh)

	//systemUserController := systemuser.NewSystemUserController(cc)
	//go systemUserController.Run(stopCh)

	//podTaskController := podtask.NewPodTaskController(cc)
	//go podTaskController.Run(stopCh)

	//websiteController := website.NewWebsiteController(cc)
	//go websiteController.Run(stopCh)

	glog.Infof("Starting informers")
	//go cc.Sitepods().StartInformer(stopCh)
	//go cc.PVClaims().StartInformer(stopCh)
	//go cc.PVs().StartInformer(stopCh)
	//go cc.Pods().StartInformer(stopCh)
	//go cc.PodTasks().StartInformer(stopCh)
	//go cc.Deployments().StartInformer(stopCh)
	//go cc.ReplicaSets().StartInformer(stopCh)
	//go cc.SystemUsers().StartInformer(stopCh)
	//go cc.ConfigMaps().StartInformer(stopCh)
	//go cc.Clusters().StartInformer(stopCh)
	//go cc.AppComps().StartInformer(stopCh)
	//go cc.Websites().StartInformer(stopCh)
	go cc.SitepodUsers().StartInformer(stopCh)
	glog.Infof("Started informers")
	glog.Info("Started simple system")
	<-stopCh
	glog.Infof("Simple system stopped")
}
