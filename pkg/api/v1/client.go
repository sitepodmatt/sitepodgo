package v1

import "k8s.io/kubernetes/pkg/client/restclient"
import "k8s.io/kubernetes/pkg/labels"
import "k8s.io/kubernetes/pkg/runtime"

type Client struct {
	rc *restclient.RESTClient
}

func (c *Client) Delete(kindName string, labels labels.Selector) (runtime.Object, error) {
	r := c.rc.Delete().Resource(kindName).LabelsSelectorParam(labels).Do()
	return r.Get()
}
