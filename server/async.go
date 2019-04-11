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
	"github.com/palantir/duo-bot/state"
	"github.com/pkg/errors"
)

const (
	// We'll probably never hit this - this is just here in case we have an I/O issue in our long-poll
	// to DUO's auth_status endpoint, because DUO should timeout in 60s
	asyncTimeout = 70 * time.Second
)

type duoTXNTracker struct {
	key    string
	txnid  string
	ts     time.Time
	logger *log.Entry
	server *Server
}

func (s *Server) newDuoTXNTracker(key string, txnid string, ts time.Time, logger *log.Entry) *duoTXNTracker {
	logger = logger.WithFields(log.Fields{
		"TXNID": txnid,
	})

	d := duoTXNTracker{
		key:    key,
		txnid:  txnid,
		ts:     ts,
		logger: logger,
		server: s,
	}

	return &d
}

func (d *duoTXNTracker) asyncHelper() {
	ok := d.waitForAuth()
	if ok {
		d.logger.Debug("Got success from DUO, attempting to mark prompt as success")
		err := d.server.state[d.key].TryAllow(d.ts)
		if err != nil {
			d.logger.Error(err)
		}
	} else {
		d.logger.Debug("Got deny from DUO, marking prompt as deny")
		d.server.state[d.key].Deny()
	}
}

func (d *duoTXNTracker) waitForAuth() bool {
	timer := time.NewTimer(asyncTimeout)
	resChan := make(chan state.PromptStatus)

	defer timer.Stop()

	go d.authStatus(resChan)

	// Keep trying until we're timed out or got a result or got an error
	for {
		select {
		case <-timer.C:
			d.logger.Error("Timed-out waiting for auth_status to return")
			return false
		case curRes := <-resChan:
			if curRes == state.StatusPending {
				d.logger.Debug("Still waiting in waitForAuth")
				continue
			}

			return curRes == state.StatusAllowed
		}
	}
}

// This function will "long-poll" to mimic how the DUO endpoint we're querying works
// that is, something will be put onto the resChan only when I get something new returned from duo.AuthStatus
func (d *duoTXNTracker) authStatus(resChan chan state.PromptStatus) {
	for {
		log.Debug("Initiating call to DUO's auth_status endpoint")
		res, err := d.server.duo.AuthStatus(d.txnid)
		if err != nil {
			resChan <- state.StatusDenied
			d.logger.Error(errors.Wrap(err, "Error checking DUO auth status"))
			return
		}

		if res == nil {
			resChan <- state.StatusDenied
			d.logger.Error(errors.New("empty response from auth_status"))
			return
		}

		if res.Stat != "OK" {
			resChan <- state.StatusDenied
			d.logger.Error(errors.Errorf("Error reported by auth_status: %s", res.Response.Status_Msg))
			return
		}

		// The only true condition - the async request has been accepted
		if res.Response.Result == "allow" {
			resChan <- state.StatusAllowed
			return
		}

		// We're waiting, but haven't been rejected yet
		if res.Response.Result == "waiting" {
			d.logger.Infof("Got waiting for reason '%s' from auth_status", res.Response.Status_Msg)
			resChan <- state.StatusPending
		} else {
			// Fail closed, an explicit deny whould hit this
			resChan <- state.StatusDenied
			return
		}
	}
}
