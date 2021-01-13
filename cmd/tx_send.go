/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/gagarinchain/network/run"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var (
	ScenarioPath string
	RpcPath      string
	SendersPath  string
)
var txSendCmd = &cobra.Command{
	Use:   "tx_send",
	Short: `Sends transactions`,
	Long:  "Is used to send transactions to the network",
	Run: func(cmd *cobra.Command, args []string) {
		run.CreateExecution(&run.Settings{
			ScenarioPath: ScenarioPath,
			RpcPath:      RpcPath,
			SendersPath:  SendersPath,
		}).Execute()
	},
}

func init() {
	rootCmd.AddCommand(txSendCmd)
	txSendCmd.PersistentFlags().StringVar(&ScenarioPath, "scenario.path", "", "Path to scenario.yaml file")
	txSendCmd.PersistentFlags().StringVar(&RpcPath, "rpc.address", "", "Rpc service path")
	txSendCmd.PersistentFlags().StringVar(&SendersPath, "senders.path", "", "Path to senders.yaml file")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
