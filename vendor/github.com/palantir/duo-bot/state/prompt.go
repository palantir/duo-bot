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

package state

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// Don't accept any results older than 10 minutes
const maxAge = 10 * time.Minute

// PromptStatus is an int used as an enum to indicate the status of a particular prompt
type PromptStatus int

const (
	// StatusAllowed means the prompt is allowed
	StatusAllowed PromptStatus = iota
	// StatusDenied means the prompt has been denied
	StatusDenied
	// StatusPending means the prompt is still outstanding and we don't yet know if it's allowed or denied
	StatusPending
)

// Prompt object holds information about an MFA prompt
type Prompt struct {
	created time.Time
	user    string
	status  PromptStatus
}

// NewPrompt returns a Prompt object, setting valid to nil because the request is still in flight
func NewPrompt(created time.Time, user string) *Prompt {
	p := Prompt{
		created: created,
		user:    user,
		status:  StatusPending,
	}

	return &p
}

// Deny marks a prompt as denied
func (p *Prompt) Deny() {
	p.status = StatusDenied
}

// Don't let outsiders call this directly, they have to call TryAllow
func (p *Prompt) allow() {
	p.status = StatusAllowed
}

// TryAllow will mark the MFA prompt as allowed, iff the time given matches the time of the prompt
// If there is a time mismatch, the prompt will be marked as denied
func (p *Prompt) TryAllow(created time.Time) error {
	// Created time I'm checking on is the same one in state, so we're good
	if p.created == created {
		p.allow()
		return nil
	}

	// There must have been an attempted race on validations, so fail closed
	p.Deny()
	return errors.Errorf("created time for this request (%v) doesn't match pending time in state (%v), rejecting", created, p.created)
}

// IsValid returns whether or not the prompt is valid, as well as a string giving more context
// passing-in a user is optional - if you don't, success doesn't depend on who accepted the MFA
func (p *Prompt) IsValid(user string) (bool, string) {
	now := time.Now()
	fmtTime := p.created.UTC().Format(time.RFC822)

	if now.Sub(p.created) > maxAge {
		return false, fmt.Sprintf("Last record created at %s is too old, try again\n", fmtTime)
	}

	if p.status == StatusPending {
		return false, fmt.Sprintf("Pending request out for user %s created at %s, please try again\n", p.user, fmtTime)
	}

	if user != "" && user != p.user {
		return false, fmt.Sprintf("Only record for key is for user %s at %s (you required user %s)\n", p.user, fmtTime, user)
	}

	if p.status == StatusAllowed {
		return true, fmt.Sprintf("Record created at %s for user %s is accepted and valid\n", fmtTime, p.user)
	}

	return false, fmt.Sprintf("Record created at %s for user %s denied or failed\n", fmtTime, p.user)
}
