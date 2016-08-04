package shared

import (
	//"github.com/golang/glog"

	"github.com/golang/glog"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/util/workqueue"
	cc "sitepod.io/sitepod/pkg/client"
	"time"
)

var (
	RetryDelay time.Duration = 200 * time.Millisecond
)

type Syncer interface {
	HasSynced() bool
}

type SimpleController struct {
	Client           *cc.Client
	waitForInformers []Syncer
	SyncFunc         func(string) error
	DeleteFunc       func(string) error
	queue            workqueue.DelayingInterface
}

func NewSimpleController(client *cc.Client, waitForInformers []Syncer,
	syncFunc func(string) error, deleteFunc func(string) error) *SimpleController {
	workQueue := workqueue.NewDelayingQueue()
	return &SimpleController{client, waitForInformers, syncFunc, deleteFunc, workQueue}
}

func (c *SimpleController) Run(stopCh <-chan struct{}) {
	go c.worker()
	<-stopCh
	c.queue.ShutDown()
}

func (c *SimpleController) EnqueueUpdate(key string) {
	glog.Infof("Enqueuing for update %s", key)
	c.queue.Add(addUpdateRequest{key})
}
func (c *SimpleController) EnqueueDelete(key string) {
	glog.Infof("Enqueuing for delete %s", key)
	c.queue.Add(deleteRequest{key})
}

func (c *SimpleController) HasSynced() bool {
	// TODO: What is this supposed to do for an aggregating controller?
	return true
}

type addUpdateRequest struct {
	key string
}

type deleteRequest struct {
	key string
}

func (c *SimpleController) worker() {
	c.WaitReady()

	glog.Infof("Processing queue")
	for {
		func() {
			item, quit := c.queue.Get()
			if quit {
				return
			}
			defer c.queue.Done(item)

			func() {

				defer func() {
					if r := recover(); r != nil {
						if errResult, ok := r.(*kerrors.StatusError); ok {
							if errResult.ErrStatus.Reason == "Conflict" {
								glog.Infof("Due to conflict requeueing %+v", item)
								c.queue.AddAfter(item, RetryDelay)
							}
						}
						glog.Errorf("Panic processing %+v: %+v", item, r)
					}
				}()

				switch item.(type) {
				case addUpdateRequest:
					req := item.(addUpdateRequest)
					glog.Infof("Processing update for %s", req.key)
					if c.SyncFunc != nil {
						err := c.SyncFunc(req.key)
						if err != nil {
							glog.Errorf("Rejected processing %s: %s", req.key, err)
						}
					}
				case deleteRequest:
					req := item.(deleteRequest)
					glog.Infof("Processing delete for %s", req.key)
					//TODO process by key
				default:
				}

			}()
		}()

	}
}

func (c *SimpleController) WaitReady() {
	for {
		allReady := true
		glog.Infof("Waiting for dependencies to be ready")
		for _, di := range c.waitForInformers {
			if !di.HasSynced() {
				allReady = false
				break
			}
		}
		if allReady {
			glog.Infof("Dependencies all ready")
			break
		} else {
			time.Sleep(RetryDelay)
		}
	}
}

type DependentResourcesNotReady struct {
	Message string
}

func (e DependentResourcesNotReady) Error() string {
	return e.Message
}

type DependentConfigNotValid struct {
	Message string
}

func (e DependentConfigNotValid) Error() string {
	return e.Message
}

//func (c *SitepodController) deleteSitepod(key string) {

//deploymentObjs, err := c.deploymentInformer.GetIndexer().ByIndex("sitepod", key)

//for _, deploymentObj := range deploymentObjs {
//doneDeployment := deploymentObj.(*ext_api.Deployment)
////err = c.deploymentDeleter(doneDeployment)
//glog.Infof("Deleting deployment %s", doneDeployment.Name)
//if doneDeployment.Spec.Replicas != 0 {
//glog.Infof("Setting replicas to 0 for %s", doneDeployment.Name)
//doneDeployment.Spec.Replicas = 0
//_, err = c.deploymentUpdater(doneDeployment)
//if err != nil {
//glog.Errorf("Unable to set replicates to 0 on deployment: %+v", err)
//}
//c.queue.AddAfter(deleteSitepodRequest{key}, RetryDelay)
//return
//} else {
//if doneDeployment.Status.Replicas != 0 {
//// TODO use delayed workqueue
//glog.Infof("Replicates not yet 0")
//c.queue.AddAfter(deleteSitepodRequest{key}, RetryDelay)
//} else {
//glog.Infof("Replicates now yet 0")

//err := c.deploymentDeleter(doneDeployment)
//if err != nil {
//glog.Errorf("Unable to delete deployment")
//return
//}

//selector, err := unversioned.LabelSelectorAsSelector(doneDeployment.Spec.Selector)
////c.rsSet(labels.Newre
//rsObjs, err := c.rsFilter(selector)
//if err != nil {
//glog.Errorf("Unable to get replica sets %v", err)
//return
//}

//rsList := rsObjs.(*ext_api.ReplicaSetList)
//for _, rsObj := range rsList.Items {
//c.rsDeleter(&rsObj)
//}

//}
//}

//}
//glog.Infof("Deleteing related system users")
//req, err := labels.NewRequirement("sitepod", labels.EqualsOperator, sets.NewString(key))
//if err != nil {
//panic(err)
//}
//sitepodMatcher := labels.NewSelector().Add(*req)
////TODO: figure out where to host this list
//sitepodResources := []string{"systemusers", "serviceinstances"}

//for _, sitepodResource := range sitepodResources {
//res := c.rc.Delete().Resource("systemusers").Namespace("default").LabelsSelectorParam(sitepodMatcher).Do()

//if err = res.Error(); err != nil {
//glog.Errorf("Unable to delete %s: %+v", sitepodResource, err)
//c.queue.AddAfter(deleteSitepodRequest{key}, RetryDelay)
//}
//}
//}
