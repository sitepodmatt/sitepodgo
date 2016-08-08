package shared

import (
	"github.com/golang/glog"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/util/workqueue"
	"reflect"
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

func (c *SimpleController) EnqueueUpdateAfter(key string, seconds int) {
	glog.Infof("Enqueuing for update %s after %d seconds", key, seconds)
	c.queue.AddAfter(addUpdateRequest{key}, time.Second*time.Duration(seconds))
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

				var err error
				switch item.(type) {
				case addUpdateRequest:
					req := item.(addUpdateRequest)
					glog.Infof("Processing update for %s", req.key)
					if c.SyncFunc != nil {
						err = c.SyncFunc(req.key)
					} else {
						glog.Infof("No sync function for controller")
					}
				case deleteRequest:
					req := item.(deleteRequest)
					glog.Infof("Processing delete for %s", req.key)
					if c.DeleteFunc != nil {
						err = c.DeleteFunc(req.key)
					} else {
						glog.Infof("No delete function for controller")
					}
				default:
				}
				if err != nil {
					if retryable, ok := err.(Retryable); ok && retryable.Retry() {
						glog.Infof("Queueing for retry %+v due to %s", item, err)
						c.queue.AddAfter(item, RetryDelay)
					} else {
						glog.Errorf("Rejected processing %+v: %s", item, err)
					}
				} else {
					glog.Infof("Completed %+v", item)
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
				glog.Infof("Informer not ready: %s", reflect.TypeOf(di).Elem().Name())
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

type Retryable interface {
	Retry() bool
}

type ConditionsNotReady struct {
	Message string
}

func (e ConditionsNotReady) Error() string {
	return e.Message
}

func (e ConditionsNotReady) Retry() bool {
	return true
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
