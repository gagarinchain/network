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
	"github.com/gagarinchain/common"
	"github.com/gagarinchain/network/run"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start network",
	Long:  "Is used to start network node",
	Run: func(cmd *cobra.Command, args []string) {
		//getString := viper.GetString("Hotstuff.SeedPath")
		////web3 := viper.GetString("web3")
		//getBool, _ := cmd.PersistentFlags().GetBool("web3")

		s := &common.Settings{}
		if err := viper.Unmarshal(s); err != nil {
			return
		}

		run.Start(s)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	startCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Path to settings.yaml file(default is $HOME/settings.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	startCmd.Flags().BoolP("toggle", "t", false, "Toggle")

	startCmd.PersistentFlags().StringP("rpc.type", "r", viper.GetString("rpc.type"), "Enables rpc service of given type, 'grpc' and 'web3' are supported now")
	startCmd.PersistentFlags().StringP("rpc.host", "j", viper.GetString("rpc.host"), "Enables grpc service on this address")
	startCmd.PersistentFlags().StringP("rpc.port", "o", viper.GetString("rpc.port"), "Enables grpc service on this port")
	startCmd.PersistentFlags().StringP("me", "m", viper.GetString("hotstuff.me"), "Current node index in committee")
	startCmd.PersistentFlags().StringP("extaddr", "a", viper.GetString("network.ExtAddr"), "Current node external address for NAT lookup")
	startCmd.PersistentFlags().StringP("plugins.address", "p", viper.GetString("plugins.address"), "Plugin service address")
	startCmd.PersistentFlags().StringArrayP("plugins.interfaces", "i", viper.GetStringSlice("plugins.interfaces"), "Plugin interfaces")

	if err := viper.BindPFlag("hotstuff.Me", startCmd.PersistentFlags().Lookup("me")); err != nil {
		println(err.Error())
	}
	if err := viper.BindEnv("hotstuff.Me", "GN_INDEX"); err != nil {
		println(err.Error())
	}
	if err := viper.BindPFlag("Network.ExtAddr", startCmd.PersistentFlags().Lookup("extaddr")); err != nil {
		println(err.Error())
	}
	if err := viper.BindEnv("Network.ExtAddr", "GN_EXTADDR"); err != nil {
		println(err.Error())
	}
	if err := viper.BindPFlag("Plugins.Address", startCmd.PersistentFlags().Lookup("plugins.address")); err != nil {
		println(err.Error())
	}
	if err := viper.BindEnv("Plugins.Address", "GN_PLUGIN"); err != nil {
		println(err.Error())
	}
	if err := viper.BindEnv("Plugins.Interfaces", "GN_PLUGIN_I"); err != nil {
		println(err.Error())
	}
	if err := viper.BindPFlag("Rpc.Type", startCmd.PersistentFlags().Lookup("rpc.type")); err != nil {
		println(err.Error())
	}
	if err := viper.BindEnv("Rpc.Type", "GN_RPC_TYPE"); err != nil {
		println(err.Error())
	}
	if err := viper.BindPFlag("Rpc.Host", startCmd.PersistentFlags().Lookup("rpc.host")); err != nil {
		println(err.Error())
	}
	if err := viper.BindEnv("Rpc.Host", "GN_RPC_HOST"); err != nil {
		println(err.Error())
	}
	if err := viper.BindPFlag("Rpc.Port", startCmd.PersistentFlags().Lookup("rpc.port")); err != nil {
		println(err.Error())
	}
	if err := viper.BindEnv("Rpc.Port", "GN_RPC_PORT"); err != nil {
		println(err.Error())
	}

	viper.SetDefault("Rpc.MaxConcurrentStreams", 10)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
