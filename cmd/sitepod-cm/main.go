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
	glog.Info("Starting sitepod single-node minion")

	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UTC().UnixNano())

	cmd.Execute()
}
