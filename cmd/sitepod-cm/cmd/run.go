package cmd

import (
	"github.com/spf13/cobra"
	"sitepod.io/sitepod/pkg/system"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run in server mode",
	Run: func(cmd *cobra.Command, args []string) {

		stopCh := make(chan struct{})
		c := system.NewSimpleSystem()
		c.Run(stopCh)
	},
}

func init() {
	RootCmd.AddCommand(runCmd)
	runCmd.PersistentFlags().String("apiserver", "http://127.0.0.1:8080", "root URL to api-server e.g. https://127.0.0.1:6443")
	runCmd.PersistentFlags().String("namespace", "default", "namespace to operate on")
}
