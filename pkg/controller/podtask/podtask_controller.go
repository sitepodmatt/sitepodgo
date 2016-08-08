package podtask

import (
	"bytes"
	"fmt"
	//"time"

	"k8s.io/kubernetes/pkg/client/unversioned/remotecommand"
	remotecommandserver "k8s.io/kubernetes/pkg/kubelet/server/remotecommand"

	"github.com/golang/glog"
	k8s_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/controller/framework"
	//"sitepod.io/sitepod/pkg/api/v1"
	cc "sitepod.io/sitepod/pkg/client"
	. "sitepod.io/sitepod/pkg/controller/shared"
)

type PodTaskController struct {
	SimpleController
}

func NewPodTaskController(client *cc.Client) framework.ControllerInterface {

	glog.Infof("Creating podtask controller")
	c := &PodTaskController{*NewSimpleController(client, []Syncer{client.PodTasks(),
		client.Pods()}, nil, nil)}

	c.SyncFunc = c.ProcessUpdate

	client.PodTasks().AddInformerHandlers(framework.ResourceEventHandlerFuncs{
		AddFunc:    c.QueueAdd,
		UpdateFunc: c.QueueUpdate,
		DeleteFunc: c.QueueDelete,
	})
	return c

}

func (c *PodTaskController) QueueAdd(item interface{}) {
	c.EnqueueUpdate(c.Client.PodTasks().KeyOf(item))
}

func (c *PodTaskController) QueueUpdate(old interface{}, cur interface{}) {
	if !c.Client.PodTasks().DeepEqual(old, cur) {
		c.EnqueueUpdate(c.Client.PodTasks().KeyOf(cur))
	}
}

func (c *PodTaskController) QueueDelete(deleted interface{}) {
	c.EnqueueDelete(c.Client.PodTasks().KeyOf(deleted))
}

func (c *PodTaskController) ProcessUpdate(key string) error {

	podTask, exists := c.Client.PodTasks().MaybeGetByKey(key)
	_ = podTask

	if !exists {
		glog.Infof("PodTask %s not longer available. Presume this has since been deleted", key)
		return nil
	}

	// TODO expect gc to clean these up eventually
	if podTask.Status.Completed == true || podTask.Status.Attempts < podTask.Spec.MaxAttempts {
		glog.Info("Skipping podtask %s", key)
	}

	stdOut, stdErr, err := c.Execute(podTask.Spec.PodName, podTask.Spec.ContainerName, podTask.Spec.Command)

	if err != nil {
		// UPDATE status retried
		podTask.Status.Attempts = podTask.Status.Attempts + 1
		podTask.Status.ExitCode = 2                     //how do we get this?
		podTask.Status.StdErr = fmt.Sprintf("%+v", err) // eek
		podTask.Status.StdOut = ""
		c.Client.PodTasks().Update(podTask)
		c.EnqueueUpdateAfter(key, 15)
	} else {

		glog.Infof("PodTask %s succeeded. Stdout: %s, Stderr: %s", podTask.Name, stdOut, stdErr)

		podTask.Status.Attempts = podTask.Status.Attempts + 1
		podTask.Status.Completed = true
		podTask.Status.ExitCode = 0
		podTask.Status.StdOut = stdOut
		podTask.Status.StdErr = stdErr
		c.Client.PodTasks().Update(podTask)
	}

	return nil

}

func (c *PodTaskController) Execute(podName string, containerName string, command []string) (string, string, error) {

	glog.Infof("Exsecuting")
	req := c.Client.Pods().RestClient().Post().
		Resource("pods").
		Name(podName).
		Namespace("default"). //TODO inject
		SubResource("exec").
		Param("container", containerName)

	req.VersionedParams(&k8s_api.PodExecOptions{
		Container: containerName,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, k8s_api.ParameterCodec)

	exec, err := remotecommand.NewExecutor(c.Client.Pods().RestClientConfig(), "POST", req.URL())

	if err != nil {
		return "", "", err
	}

	stdout := bytes.NewBuffer([]byte{})
	stderr := bytes.NewBuffer([]byte{})

	err = exec.Stream(remotecommand.StreamOptions{
		SupportedProtocols: remotecommandserver.SupportedStreamingProtocols,
		Stdin:              nil,
		Stdout:             stdout,
		Stderr:             stderr,
		Tty:                false,
		TerminalSizeQueue:  nil,
	})

	if err != nil {
		return "", "", err
	}

	return stdout.String(), stderr.String(), err
}