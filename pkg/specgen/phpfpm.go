package specgen

import (
	"sitepod.io/sitepod/pkg/api/v1"
)

func SpecGenPHPFPM(obj interface{}) error {
	ac := obj.(*v1.Appcomponent)
	ac.Spec.Type = "phpfpm"
	ac.Spec.DisplayName = "PHP FPM Server"
	ac.Spec.Description = "PHP FPM Server"
	ac.Spec.Image = "sitepod/phpfpm"
	ac.Spec.ImageVersion = "latest"

	ac.Spec.MountHome = true
	ac.Spec.MountEtcs = true
	ac.Spec.MountTemp = true

	ac.Spec.Expose = true
	ac.Spec.ExposePort = 9090
	ac.Spec.ExposeExternally = true

	return nil
}
