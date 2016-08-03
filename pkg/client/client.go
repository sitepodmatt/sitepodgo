package client

import "sitepod.io/sitepod/pkg/api/v1"

type Client struct {
}

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" SitepodClient(v1.Sitepod)

func (c *Client) Sitepods() {

}
