package sitepod

// Sitepod controller respond to additions/updates/deletions of sitepod resource type
// For new sitepods it provisions a linked (by label) persistent volume and deployment resource.
// For deletion performs similar action to kubectl reaper, change desired replicas to 0, waits
// and then removes the deployment and related replica sets.

import (
	"fmt"

	. "github.com/ahmetalpbalkan/go-linq"
	"github.com/golang/glog"
	k8s_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/controller/framework"
	"sitepod.io/sitepod/pkg/api/v1"
	cc "sitepod.io/sitepod/pkg/client"
	. "sitepod.io/sitepod/pkg/controller/shared"
)

type SitepodController struct {
	*SimpleController
}

func NewSitepodController(client *cc.Client) framework.ControllerInterface {

	sc := &SitepodController{NewSimpleController(client, []Syncer{}, nil, nil)}
	client.Sitepods().AddInformerHandlers(framework.ResourceEventHandlerFuncs{
		AddFunc:    nil,
		UpdateFunc: nil,
		DeleteFunc,
	})
}

func (sc *SitepodController) AddFunc(item interface{}) {
	sc.EnqueueUpdate(client.Sitepods().KeyOf(item))
}

func (sc *SitepodController) UpdateFunc(old interface{}, cur interface{}) {
	if !client.Sitepods().DeepEqual(old, cur) {
		sc.EnqueueUpdate(client.Sitepods().KeyOf(cur))
	}
}

func (sc *SitepodController) DeleteFunc(deleted interface{}) {
	sc.EnqueueDelete(client.Sitepods().KeyOf(cur))
}

func testSync(c *cc.Client, key string) error {

	sitepod, exists := c.Sitepods().MaybeGetByKey(key)

	if !exists {
		glog.Infof("Sitepod %s not longer available. Presume this has since been deleted", key)
		return nil
	}

	sitepodKey := string(sitepod.UID)
	_ = sitepodKey

	defaultPvc := sitepod.Spec.VolumeClaims[0]

	pvClaim, exists := c.PVClaims().MaybeGetByKey(defaultPvc)

	if !exists {
		return DependentResourcesNotReady{"PVC does not yet exist."}
	}

	if len(pvClaim.Spec.VolumeName) == 0 {
		return DependentResourcesNotReady{fmt.Sprintf("PVC %s is not yet bound to a PV", defaultPvc)}
	}

	pv := c.PVs().GetByKey(pvClaim.Spec.VolumeName)

	isHostPath := pv.Spec.HostPath != nil
	var pinnedHost string
	if isHostPath {
		if pinnedHost = pv.Annotations["sitepod.io/pinnedhost"]; len(pinnedHost) == 0 {
			return DependentConfigNotValid{"No pinned host specified for host local storage"}
		}
	}

	deployment, exists := c.Deployments().MaybeSingleBySitepodKey(sitepodKey)

	if !exists {
		//TODO change NewForSitepod(sitepodKey)
		deployment = c.Deployments().NewEmpty()
	}

	labels := make(map[string]string)
	labels["sitepod"] = sitepodKey

	deployment.Spec.Replicas = 1
	///deployment.Spec.Selector = &unversioned.LabelSelector{MatchLabels: labels}

	if isHostPath {
		deployment.Spec.Template.Spec.NodeName = pinnedHost
	}

	smExists, _ := From(deployment.Spec.Template.Spec.Containers).Where(func(s T) (bool, error) {
		return (s.(k8s_api.Container).Name == "sitepod-manager"), nil
	}).Any()

	if !smExists {
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers,
			k8s_api.Container{
				Name:  "sitepod-manager",
				Image: "gcr.io/google_containers/pause:2.0",
			})
	}

	deployment.Spec.Template.GenerateName = "sitepod-pod-"
	deployment.Spec.Template.Labels = labels
	//TODO add labels don't replace
	deployment.Labels = labels

	c.Deployments().UpdateOrAdd(deployment)

	return nil
}
