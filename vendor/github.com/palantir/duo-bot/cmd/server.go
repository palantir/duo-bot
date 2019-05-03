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
	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/palantir/duo-bot/server"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run duo-bot as a server",
	Long:  `Run duo-bot in server mode, the only mode.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.SetLevel(logLevelFromViper())

		log.Debug("server called")
		if log.GetLevel() == log.DebugLevel {
			cmd.DebugFlags()
			viper.Debug()
		}

		serverAddr := viper.GetString("server.addr")
		if serverAddr == "" {
			log.Fatal("Server addr not set, pass it in with -a")
		}

		duoHost := viper.GetString("duo.host")
		if duoHost == "" {
			log.Fatal("duo.host not set in config")
		}

		duoIkey := viper.GetString("duo.ikey")
		if duoIkey == "" {
			log.Fatal("duo.ikey not set in config")
		}

		duoSkey := viper.GetString("duo.skey")
		if duoSkey == "" {
			log.Fatal("duo.skey not set in config")
		}

		log.Debugf("%t %t", viper.Get("server.addr"), version)

		srv, err := server.New(serverAddr, version, duoHost, duoIkey, duoSkey)

		if err != nil {
			log.Fatal(err)
		} else {
			srv.Start()
		}
	},
}

func init() {
	RootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringP("addr", "a", "", "Addr to run server on in host:port form, or :port form.")
	err := viper.BindPFlag("server.addr", serverCmd.Flags().Lookup("addr"))
	if err != nil {
		log.Fatal(errors.Wrap(err, "Binding PFlag 'addr' to viper var 'server.addr' failed: %s"))
	}
}
