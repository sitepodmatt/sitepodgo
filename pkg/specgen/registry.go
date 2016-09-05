package specgen

type SpecFn func(obj interface{}) error

var specMap map[string]SpecFn

func init() {
	specMap = make(map[string]SpecFn)
	specMap["ssh-server"] = SpecGenSSHServer
	specMap["web-server"] = SpecGenNginxServer
	specMap["php-fpm"] = SpecGenPHPFPM
}

func RegisterSpecGen(key string, fn SpecFn) {
	specMap[key] = fn
}

func Lookup(key string) SpecFn {
	return specMap[key]
}
