// Copyright 2021-2023
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package gotasty provides an idiomatic go interface to the tastytrade
// Open API. It implements session management, account information,
// order execution, and streaming quotes.

package gotasty

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

const (
	userAgent = "go-tasty/1.0.0 (https://github.com/penny-vault/go-tasty)"

	sandboxApiURL = "https://api.cert.tastyworks.com"
	apiURL        = "https://api.tastyworks.com"

	sandboxAccountStreamerURL = "wss://streamer.cert.tastyworks.com"
	accountStreamerURL        = "wss://streamer.tastyworks.com"
)

// Session stores user credentials and enables users to make authenticated
// requests of the tastytrade Open API. Sessions are safe for concurrent
// use in multiple goroutines.
type Session struct {
	AuthenticatedOn     time.Time // time the session was first authenticated
	ExpiresOn           time.Time // time when the session token will expire
	RememberMeExpiresOn time.Time // time when the remember-me token will expire

	Name       string
	Nickname   string
	Email      string
	ExternalID string
	Username   string

	ApiURL             string // Base URL of the api, changes based on production vs sandbox environment
	AccountStreamerURL string // Base URL of websocket for account streaming data

	Token *atomic.Value // Session token - valid for 24 hours

	// Remember token - can be exchanged for a new session token. Each
	// remember token can be used exactly once and expire after 28 days
	RememberToken *atomic.Value
}

// SessionOpts provide additional settings when creating a new tastytrade Open API session
type SessionOpts struct {
	// request a remember-me token which enables the API to refresh session
	// tokens for up-to 28 days
	RememberMe bool

	// use the tastytrade Open API sandbox environment for testing
	Sandbox bool

	// create a go routine that will automatically refresh the session when it expires
	EnableAutomaticRefresh bool

	// enable debug mode which prints the status of each request
	Debug bool
}

// NewSession obtains a session token and optionally a remember-me token from the
// tastytrade Open API. If you want sessions to be refreshed after they expire,
// set the `SessionOpts.RememberMe` and `SessionOpts.EnableAutomaticRefresh` options.
func NewSession(login, password string, opts ...SessionOpts) (*Session, error) {
	var opt SessionOpts
	if len(opts) > 0 {
		opt = opts[0]
	}

	client := resty.New()

	client.SetDebug(opt.Debug)

	client.SetHeaders(map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   userAgent,
	})

	url := apiURL
	accountStreamerURL := accountStreamerURL
	if opt.Sandbox {
		url = sandboxApiURL
		accountStreamerURL = sandboxAccountStreamerURL
	}

	client.SetBaseURL(url)

	resp, err := client.R().
		SetBody(User{Username: login, Password: password, RememberMe: opt.RememberMe}).
		Post("/sessions")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("%s: %s", resp.Status(), resp.Body())
	}

	session := &Session{
		AccountStreamerURL: accountStreamerURL,
		ApiURL:             url,

		AuthenticatedOn: resp.ReceivedAt(),
		ExpiresOn:       resp.ReceivedAt().Add(24 * time.Hour),

		Username: login,

		Token:         &atomic.Value{},
		RememberToken: &atomic.Value{},
	}

	body := string(resp.Body())
	session.Token.Store(gjson.Get(body, "data.session-token").Str)

	if opt.RememberMe {
		session.RememberMeExpiresOn = resp.ReceivedAt().Add(28 * 24 * time.Hour)
		session.RememberToken.Store(gjson.Get(body, "data.session-token").Str)
	}

	session.Name = gjson.Get(body, "data.user.name").Str
	session.Nickname = gjson.Get(body, "data.user.nickname").Str
	session.Email = gjson.Get(body, "data.user.email").Str
	session.ExternalID = gjson.Get(body, "data.user.external-id").Str

	return session, nil
}
