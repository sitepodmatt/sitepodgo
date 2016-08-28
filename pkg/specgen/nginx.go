package specgen

import (
	"sitepod.io/sitepod/pkg/api/v1"
)

func SpecGenNginxServer(obj interface{}) error {
	ac := obj.(*v1.Appcomponent)
	ac.Spec.Type = "webserver"
	ac.Spec.DisplayName = "Nginx Web Server"
	ac.Spec.Description = "Nginx Web Server"
	ac.Spec.Image = "sitepod/nginx"
	ac.Spec.ImageVersion = "latest"

	ac.Spec.MountHome = true
	ac.Spec.MountEtcs = true

	ac.Spec.Expose = true
	ac.Spec.ExposePort = 80
	ac.Spec.ExposeExternally = true

	return nil
}
