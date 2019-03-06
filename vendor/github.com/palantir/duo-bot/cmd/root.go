// Copyright 2017 Palantir Technologies, Inc.
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
	"fmt"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	validator "gopkg.in/go-playground/validator.v9"
)

var (
	cfgFile     string
	version     = "unspecified"
	versionFlag bool
	validate    *validator.Validate
)

func init() {
	cobra.OnInitialize()
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "duo-bot",
	Short: "An app for tracking DUO prompts",
	Long:  `An app that tracks a DUO prompt per arbitrary key, and who accepted the prompt`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if versionFlag {
			fmt.Println(version)
			os.Exit(0)
		}
		switch cmdName := cmd.Name(); cmdName {
		case "server":
			log.SetFormatter(&log.JSONFormatter{})
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("You must provide a command")
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	validate = validator.New()

	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is duo-bot.yaml)")
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable Verbose debugging output")
	err := viper.BindPFlag("verbose", RootCmd.PersistentFlags().Lookup("verbose"))
	if err != nil {
		log.Fatal(errors.Wrap(err, "Error binding verbose flag"))
	}

	RootCmd.PersistentFlags().BoolVar(&versionFlag, "version", false, "Print Version and exit")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile == "" {
		log.Fatal("You must specify a config file")
	}

	viper.SetConfigFile(cfgFile)

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
	} else {
		log.Debugf("Not using config file: %s", err)
	}
}

func logLevelFromViper() log.Level {
	if viper.GetBool("verbose") {
		return log.DebugLevel
	}
	logLevel := viper.GetString("deploy.log")
	level := strings.ToLower(logLevel)
	switch level {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	case "panic":
		return log.PanicLevel
	default:
		return log.InfoLevel
	}
}
