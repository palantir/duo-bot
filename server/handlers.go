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
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"github.com/twinj/uuid"
)

type healthCheckPayload struct {
	Healthy string `json:"healthy"`
	Version string `json:"version"`
}

// MetadataPayload object is what clients send to include
// as extra metadata in the DUO push request
type MetadataPayload struct {
	DuoPushInfo string `json:"duoPushInfo"`
}

func getLogger(key string, user string) *log.Entry {
	requestID := uuid.NewV4()

	newLogger := log.New()
	newLogger.Formatter = log.StandardLogger().Formatter
	newLogger.Level = log.StandardLogger().Level
	logger := newLogger.WithFields(log.Fields{
		"key":       key,
		"requestID": requestID.String(),
		"user":      user,
	})

	return logger
}

func (s *Server) pushHandler(c echo.Context) error {
	return s.promptHandler(c, "push")
}

func (s *Server) passcodeHandler(c echo.Context) error {
	return s.promptHandler(c, "passcode")
}

func (s *Server) smsHandler(c echo.Context) error {
	return s.promptHandler(c, "sms")
}

func (s *Server) phoneHandler(c echo.Context) error {
	return s.promptHandler(c, "phone")
}

func (s *Server) promptHandler(c echo.Context, factor string) error {
	// Check if there's a pending challenge for this key
	// Issue challenge for this key to this user
	key := c.Param("key")
	user := c.QueryParam("user")
	device := c.QueryParam("device")
	passcode := c.QueryParam("passcode")
	asyncParam := c.QueryParam("async")

	async := false
	if asyncParam == "1" {
		async = true
	}

	logger := getLogger(key, user)

	logger.Info("Clobbering previous state for key, if any")
	// Any new request clobbers any previous one and sets status to pending
	// Return a timestamp so we know we're only updating state if they match
	ts := s.resetStateForKey(key, user)

	curPrompt := s.state[key]

	meta := new(MetadataPayload)
	err := c.Bind(meta)
	if err != nil {
		msg := errors.Wrap(err, "error binding extra metadata object, skipping")
		logger.Warn(msg)
	}

	pc, err := newPromptConfig(user, factor, device, passcode, async)
	if err != nil {
		curPrompt.Deny()
		logger.Error(err.Error())
		return c.String(http.StatusBadRequest, err.Error())
	}

	logger.Info("Calling DUO prompt")
	res, err := s.prompt(pc, key, meta)
	if err != nil {
		msg := errors.Wrap(err, "Error from DUO")
		logger.Error(msg)
		curPrompt.Deny()
		return c.String(http.StatusBadRequest, msg.Error())
	}

	if pc.async {
		// We want to decorate res before returning it to the user, but we need
		// the raw TXN ID returned as well
		txnID := res
		res = fmt.Sprintf("Async prompt sent, txn ID: %s\n", res)
		// Create a goroutine to poll for change of this state
		logger.Info(res)
		dt := s.newDuoTXNTracker(key, txnID, ts, logger)
		go dt.asyncHelper()
	} else {
		res = fmt.Sprintf("Prompt successful: %s", res)
		logger.Info(res)
		err = curPrompt.TryAllow(ts)
		if err != nil {
			logger.Error(err)
			return c.String(http.StatusInternalServerError, err.Error())
		}
	}

	return c.String(http.StatusOK, res)
}

func (s *Server) checkHandler(c echo.Context) error {
	key := c.Param("key")
	user := c.QueryParam("user")

	logger := getLogger(key, user)

	valid, msg := s.isValid(key, user)

	logger.Info(msg)

	if valid {
		return c.String(http.StatusOK, msg)
	}

	return c.String(http.StatusInternalServerError, msg)
}

func (s *Server) healthHandler(c echo.Context) error {
	p := healthCheckPayload{
		Healthy: "yes",
		Version: s.version,
	}
	return c.JSON(http.StatusOK, p)
}
