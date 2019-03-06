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
	"net/url"

	log "github.com/Sirupsen/logrus"
	"github.com/duosecurity/duo_api_golang/authapi"
	"github.com/pkg/errors"
)

const (
	// Arbitrary string to display in push notifications as the type
	// https://duo.com/docs/authapi#/auth (see type under Duo Push)
	duoAuthType = "Transaction"
)

type promptConfig struct {
	user     string
	factor   string
	device   string
	passcode string
	async    bool
}

func newPromptConfig(user string, factor string, device string, passcode string, async bool) (*promptConfig, error) {
	if user == "" {
		return nil, errors.New("you must specify a user to prompt")
	}

	if factor == "" {
		factor = "push"
	}
	if device == "" {
		device = "auto"
	}

	if factor == "passcode" && passcode == "" {
		return nil, errors.New("to use factor=passcode, you must specify a passcode")
	}

	pc := promptConfig{
		user:     user,
		factor:   factor,
		device:   device,
		passcode: passcode,
		async:    async,
	}

	return &pc, nil
}

func (s *Server) newDuoAuth(pc *promptConfig, key string, meta *MetadataPayload) (*authapi.AuthResult, error) {
	log.WithFields(log.Fields{
		"factor":   pc.factor,
		"username": pc.user,
		"passcode": pc.passcode,
		"device":   pc.device,
		"async":    pc.async,
	}).Debug("Issuing DUO call")

	var options []func(*url.Values)

	// Everything needs username
	options = append(options, authapi.AuthUsername(pc.user))

	if pc.factor == "passcode" {
		options = append(options, authapi.AuthPasscode(pc.passcode))
		return s.duo.Auth(pc.factor, options...)
	}

	// Everything else involves a device, so needs at least that
	options = append(options, authapi.AuthDevice(pc.device))

	// phone and push can use this
	if pc.async {
		options = append(options, authapi.AuthAsync())
	}

	if pc.factor == "push" {
		duoPushInfo := fmt.Sprintf("Key=%s", key)

		if meta.DuoPushInfo != "" {
			duoPushInfo = fmt.Sprintf("%s&%s", duoPushInfo, meta.DuoPushInfo)
		}

		options = append(options, authapi.AuthPushinfo(duoPushInfo), authapi.AuthType(duoAuthType))
	}

	return s.duo.Auth(pc.factor, options...)
}

func (s *Server) prompt(pc *promptConfig, key string, meta *MetadataPayload) (string, error) {
	res, err := s.newDuoAuth(pc, key, meta)
	if err != nil {
		return "", errors.Wrap(err, "Error calling DUO")
	}

	// If the username specified doesn't exist
	if res.Stat != "OK" {
		return "", errors.Errorf("Error reported by DUO auth: %s\n", *res.Message)
	}

	// The only successful response from a blocking call
	if res.Response.Status == "allow" {
		return res.Response.Status_Msg, nil
	}

	// The only successful response from an async call
	if res.Response.Txid != "" {
		// Although the above case returns free-form text, this return is strict;
		// we need this ID later to check status against
		return res.Response.Txid, nil
	}

	// Fail closed
	return "", errors.Errorf("Prompt failed: %s\n", res.Response.Status_Msg)
}

func (s *Server) duoCheck() error {
	log.Info("Running initial DUO checks")

	_, err := s.duo.Ping()
	if err != nil {
		return errors.Wrap(err, "Error pinging DUO Auth API")
	}

	cr, err := s.duo.Check()
	if err != nil {
		return errors.Wrap(err, "Error checking DUO Auth API")
	}

	// Like if the ikey/skey are bad
	if cr.Stat != "OK" {
		return errors.Errorf("Error checking DUO Auth API: %s", *cr.StatResult.Message)
	}

	return nil
}
