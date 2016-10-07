// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"sitepod.io/sitepod/pkg/system"
	"sitepod.io/sitepod/pkg/util"
)

// createuserCmd represents the createuser command
var createuserCmd = &cobra.Command{
	Use:   "create-user EMAIL PASSOWRD",
	Short: "Create a new sitepod user with email as login",
	Long:  "Create a new sitepod user with email as login",
	Run: func(cmd *cobra.Command, args []string) {

		err := RunCreateUser(cmd, args)
		if err != nil {
			cmdutil.CheckErr(err)
		}
	},
}

func RunCreateUser(cmd *cobra.Command, args []string) error {

	if len(args) != 2 {
		return cmdutil.UsageError(cmd, "args should be EMAIL and PASSWORD only")
	}

	email := args[0]
	password := args[1]

	config := &system.SimpleConfig{"http://localhost:9080", "default"}

	ss := system.NewSimpleSystem(config)
	client := ss.GetClient()

	sitepodUser := client.SitepodUsers().NewEmpty()
	sitepodUser.Name = "sitepod-user-" + util.GetMD5Hash(email)
	sitepodUser.Spec.Email = email
	sitepodUser.Annotations["sitepod.io/plain-text-password"] = password

	client.SitepodUsers().Add(sitepodUser)
	return nil
}

func init() {
	adminCmd.AddCommand(createuserCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createuserCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:

}
