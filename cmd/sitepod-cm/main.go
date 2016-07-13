package main

import (
	"flag"
	"math/rand"
	"runtime"
	"time"

	"github.com/golang/glog"

	"sitepod.io/sitepod/cmd/sitepod-cm/cmd"
)

func main() {

	flag.Parse()
	flag.Set("logtostderr", "true")
	//glog.Info(flag.Lookup("v").Value.String())
	//flag.Set("v", "4")
	glog.Info("Starting sitepod single-node minion")

	glog.V(4).Info("HERE")

	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UTC().UnixNano())

	cmd.Execute()
}
