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

package server

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/duosecurity/duo_api_golang/authapi"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/palantir/duo-bot/state"
	"github.com/pkg/errors"
)

// A Server is duo-bot run in server mode, the only mode
type Server struct {
	addr    string
	version string
	duo     *authapi.AuthApi
	state   map[string]*state.Prompt
}

// Start starts the server listening on the given port
func (s *Server) Start() {
	e := echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339_nano}","id":"${id}","remote_ip":"${remote_ip}","x_forwarded_for":"${header:X-Forwarded-For}",host":"${host}",` +
			`"method":"${method}","uri":"${uri}","status":${status}, "latency":${latency},` +
			`"latency_human":"${latency_human}","bytes_in":${bytes_in},` +
			`"bytes_out":${bytes_out}}` + "\n",
	}))
	e.Use(middleware.Recover())

	err := s.duoCheck()
	if err != nil {
		e.Logger.Fatal(errors.Wrap(err, "Error running initial DUO checks"))
	}

	e.GET("/v1/health", s.healthHandler)
	e.GET("/v1/check/:key", s.checkHandler)

	e.POST("/v1/push/:key", s.pushHandler)
	e.POST("/v1/passcode/:key", s.passcodeHandler)
	e.POST("/v1/sms/:key", s.smsHandler)
	e.POST("/v1/phone/:key", s.phoneHandler)

	e.Logger.Fatal(e.Start(s.addr))
}

// New initializes a server with its config
func New(addr string, version string, duoHost string, duoIkey string, duoSkey string) (*Server, error) {
	var s Server

	s.addr = addr
	s.version = version
	duo := authapi.NewAuthApi(*duoapi.NewDuoApi(duoIkey, duoSkey, duoHost, "DUO bot"))
	s.duo = duo

	s.state = make(map[string]*state.Prompt)

	log.Debugf("Initialized DUO to point at host %s", duoHost)

	return &s, nil
}

func (s *Server) isValid(key string, user string) (bool, string) {
	p := s.state[key]
	if p != nil {
		return p.IsValid(user)
	}
	return false, "No validation record found\n"
}

func (s *Server) resetStateForKey(key string, user string) time.Time {
	ts := time.Now()
	p := state.NewPrompt(ts, user)
	s.state[key] = p
	return ts
}
