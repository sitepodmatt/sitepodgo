package main

import (
	"flag"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/golang/glog"

	"sitepod.io/sitepod/cmd/sitepod-cm/cmd"
)

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UTC().UnixNano())

	if logStdErr := flag.Lookup("logtostderr"); logStdErr != nil {
		logStdErr.DefValue = "true"
	}

	flag.Parse()

	glog.Info("Starting sitepod controller manager")

	if err := cmd.RootCmd.Execute(); err != nil {
		glog.Errorf("Command failed: %+v", err)
		os.Exit(-1)
	}
}
