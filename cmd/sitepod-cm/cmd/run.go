package cmd

import (
	"flag"
	"github.com/spf13/cobra"
	"sitepod.io/sitepod/pkg/controller"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run in server mode",
	Run: func(cmd *cobra.Command, args []string) {

		//apiServer := cmd.Flag("apiserver").Value.String()
		//namespace := cmd.Flag("namespace").Value.String()

		stopCh := make(chan struct{})
		c := controller.NewSingleNodeController(
			controller.DefaultConfig())
		c.Run(stopCh)

	},
}

func init() {
	flag.Set("v", "8")
	RootCmd.AddCommand(runCmd)
	runCmd.PersistentFlags().String("apiserver", "http://127.0.0.1:8080", "root URL to api-server e.g. https://127.0.0.1:6443")
	runCmd.PersistentFlags().String("namespace", "default", "namespace to operate on")
	runCmd.PersistentFlags().Int("v", 4, "logging level")
}
