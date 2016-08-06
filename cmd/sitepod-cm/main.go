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

	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UTC().UnixNano())

	// invert stupid glog default of logging to temp file instead
	// pass --logtostderr=false to revert to old default
	if logStdErr := flag.Lookup("logtostderr"); logStdErr != nil {
		logStdErr.DefValue = "true"
	}

	flag.Parse()

	glog.Info("Starting sitepod controller manager")

	cmd.Execute()
}
