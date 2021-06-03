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
	"fmt"
	"github.com/spf13/cobra"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gnetwork",
	Short: "Gagarin.network is Hotstuff based blockchain network",
	Long:  `Run stablecoins and develop apps on blockchain with Gagarin.network`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Path to settings.yaml file(default is $HOME/settings.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Toggle")

	rootCmd.PersistentFlags().StringP("me", "m", viper.GetString("hotstuff.me"), "Current node index in committee")
	rootCmd.PersistentFlags().StringP("extaddr", "a", viper.GetString("network.ExtAddr"), "Current node external address for NAT lookup")
	rootCmd.PersistentFlags().StringP("plugins.address", "p", viper.GetString("plugins.address"), "Plugin service address")
	rootCmd.PersistentFlags().StringArrayP("plugins.interfaces", "i", viper.GetStringSlice("plugins.interfaces"), "Plugin interfaces")

	if err := viper.BindPFlag("hotstuff.Me", rootCmd.PersistentFlags().Lookup("me")); err != nil {
		println(err.Error())
	}
	if err := viper.BindEnv("hotstuff.Me", "GN_INDEX"); err != nil {
		println(err.Error())
	}
	if err := viper.BindPFlag("Network.ExtAddr", rootCmd.PersistentFlags().Lookup("extaddr")); err != nil {
		println(err.Error())
	}
	if err := viper.BindEnv("Network.ExtAddr", "GN_EXTADDR"); err != nil {
		println(err.Error())
	}
	if err := viper.BindPFlag("Plugins.Address", rootCmd.PersistentFlags().Lookup("plugins.address")); err != nil {
		println(err.Error())
	}
	if err := viper.BindEnv("Plugins.Address", "GN_PLUGIN"); err != nil {
		println(err.Error())
	}
	if err := viper.BindEnv("Plugins.Interfaces", "GN_PLUGIN_I"); err != nil {
		println(err.Error())
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	envCfg, envFound := os.LookupEnv("GN_SETTINGS")
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else if envFound {
		viper.SetConfigFile(envCfg)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cobra_test" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName("settings.yaml")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Println(err)
	}
}
