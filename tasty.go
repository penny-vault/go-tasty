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
	"bytes"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

const (
	userAgent = "go-tasty/1.0.0 (https://github.com/penny-vault/go-tasty)"

	sandboxApiURL = "https://api.cert.tastyworks.com"
	apiURL        = "https://api.tastyworks.com"

	sandboxAccountStreamerURL = "wss://streamer.cert.tastyworks.com"
	accountStreamerURL        = "wss://streamer.tastyworks.com"
)

var (
	ErrSessionExpired       = errors.New("session token is expired")
	ErrRememberTokenExpired = errors.New("remember-me token is expired")
)

// NewSession obtains a session token and optionally a remember-me token from the
// tastytrade Open API. If you want sessions to be refreshed after they expire,
// set the `SessionOpts.RememberMe` option.
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

		RefreshLocker: &sync.Mutex{},
		Debug:         opt.Debug,
	}

	body := string(resp.Body())
	session.Token.Store(gjson.Get(body, "data.session-token").String())

	if opt.RememberMe {
		session.RememberMeExpiresOn = resp.ReceivedAt().Add(28 * 24 * time.Hour)
		session.RememberToken.Store(gjson.Get(body, "data.session-token").String())
	}

	session.Name = gjson.Get(body, "data.user.name").String()
	session.Nickname = gjson.Get(body, "data.user.nickname").String()
	session.Email = gjson.Get(body, "data.user.email").String()
	session.ExternalID = gjson.Get(body, "data.user.external-id").String()

	return session, nil
}

// NewSessionFromBytes constructs a session object from the serialized bytes
func NewSessionFromBytes(sessionData []byte) (*Session, error) {
	var data struct {
		AuthenticatedOn   int64  `json:"authenticated-on"`
		ApiURL            string `json:"url"`
		SessionToken      string `json:"token"`
		ExpiresOn         int64  `json:"expires"`
		RememberToken     string `json:"remember-token"`
		RememberExpiresOn int64  `json:"remember-expires"`

		Name       string `json:"name"`
		Nickname   string `json:"nickname"`
		Email      string `json:"email"`
		ExternalID string `json:"external-id"`
		Username   string `json:"username"`

		Debug bool `json:"debug"`
	}

	buf := bytes.NewBuffer(sessionData)
	uncompress, err := zstd.NewReader(buf)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(uncompress)

	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	session := &Session{
		Name:       data.Name,
		Nickname:   data.Nickname,
		Email:      data.Email,
		ExternalID: data.ExternalID,
		Username:   data.Username,
		Debug:      data.Debug,

		Token:         &atomic.Value{},
		RememberToken: &atomic.Value{},
	}

	if data.ApiURL == sandboxApiURL {
		session.ApiURL = sandboxApiURL
		session.AccountStreamerURL = sandboxAccountStreamerURL
	} else {
		session.ApiURL = apiURL
		session.AccountStreamerURL = accountStreamerURL
	}

	session.Token.Store(data.SessionToken)
	session.RememberToken.Store(data.RememberToken)

	session.AuthenticatedOn = time.Unix(data.AuthenticatedOn, 0)
	session.ExpiresOn = time.Unix(data.ExpiresOn, 0)
	session.RememberMeExpiresOn = time.Unix(data.RememberExpiresOn, 0)

	return session, nil
}

// Marshal serializes the Session object as a JSON string
func (session *Session) Marshal() ([]byte, error) {
	var out bytes.Buffer

	compressor, err := zstd.NewWriter(&out)
	if err != nil {
		return []byte{}, err
	}

	encoder := json.NewEncoder(compressor)

	err = encoder.Encode(struct {
		AuthenticatedOn   int64  `json:"authenticated-on"`
		ApiURL            string `json:"url"`
		SessionToken      string `json:"token"`
		ExpiresOn         int64  `json:"expires"`
		RememberToken     string `json:"remember-token"`
		RememberExpiresOn int64  `json:"remember-expires"`

		Name       string `json:"name"`
		Nickname   string `json:"nickname"`
		Email      string `json:"email"`
		ExternalID string `json:"external-id"`
		Username   string `json:"username"`

		Debug bool `json:"debug"`
	}{
		AuthenticatedOn:   session.AuthenticatedOn.Unix(),
		ApiURL:            session.ApiURL,
		SessionToken:      session.Token.Load().(string),
		ExpiresOn:         session.ExpiresOn.Unix(),
		RememberToken:     session.RememberToken.Load().(string),
		RememberExpiresOn: session.RememberMeExpiresOn.Unix(),

		Name:       session.Name,
		Nickname:   session.Nickname,
		Email:      session.Email,
		ExternalID: session.ExternalID,
		Username:   session.Username,

		Debug: session.Debug,
	})

	if err != nil {
		return []byte{}, err
	}

	return out.Bytes(), nil
}

func (session *Session) restyClient() (*resty.Client, error) {
	client := resty.New()
	client.SetBaseURL(session.ApiURL)
	client.SetHeaders(map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   userAgent,
	})
	client.SetDebug(session.Debug)

	// check if the session token is expired
	// NOTE: add a 5 minute buffer to ensure that the token doesn't expire mid-use
	if session.ExpiresOn.Before(time.Now().Add(-5 * time.Minute)) {
		session.RefreshLocker.Lock()
		defer session.RefreshLocker.Unlock()

		log.Debug().Time("TokenExpires", session.ExpiresOn).
			Time("RememberTokenExpires", session.RememberMeExpiresOn).Msg("session token is expired")

		rememberMe := session.RememberToken.Load().(string)

		// if no remember-me token available return an error
		if rememberMe == "" {
			return nil, ErrSessionExpired
		}

		// there is a remember-me token, check if it's expired
		if session.RememberMeExpiresOn.Before(time.Now()) {
			return nil, ErrRememberTokenExpired
		}

		// there is a valid remember-me token, exchange it for a session token
		resp, err := client.R().
			SetBody(User{Username: session.Username, RememberToken: session.RememberToken.Load().(string), RememberMe: true}).
			Post("/sessions")
		if err != nil {
			return nil, err
		}

		if resp.StatusCode() >= 400 {
			return nil, fmt.Errorf("%s: %s", resp.Status(), resp.Body())
		}

		body := string(resp.Body())

		session.ExpiresOn = resp.ReceivedAt().Add(24 * time.Hour)
		session.Token.Store(gjson.Get(body, "data.session-token").String())

		session.RememberMeExpiresOn = resp.ReceivedAt().Add(28 * 24 * time.Hour)
		session.RememberToken.Store(gjson.Get(body, "data.session-token").String())
	}

	client.SetHeader("Authorization", session.Token.Load().(string))

	return client, nil
}

// Accounts returns a list of accounts held by the customer
func (session *Session) Accounts() ([]*AccountInfo, error) {
	client, err := session.restyClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.R().Get("/customers/me/accounts")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("%s (accounts): %s", resp.Status(), resp.Body())
	}

	arr := gjson.Get(string(resp.Body()), "data.items").Array()
	accounts := make([]*AccountInfo, len(arr))
	for idx, acct := range arr {
		accounts[idx].AccountNumber = acct.Get("account.account-number").String()
		accounts[idx].ExternalID = acct.Get("account.external-id").String()
		accounts[idx].OpenedAt = acct.Get("account.opened-at").Time()
		accounts[idx].Nickname = acct.Get("account.nickname").String()
		accounts[idx].AccountType = acct.Get("account.account-type-name").String()
		accounts[idx].DayTraderStatus = acct.Get("account.day-trader-status").Bool()
		accounts[idx].MarginOrCash = acct.Get("account.margin-or-cash").String()
		accounts[idx].AuthorityLevel = acct.Get("authority-level").String()
		accounts[idx].IsFirmError = acct.Get("account.is-firm-error").Bool()
		accounts[idx].IsFirmProprietary = acct.Get("account.is-firm-proprietary").Bool()
		accounts[idx].IsTestDrive = acct.Get("account.is-test-drive").Bool()
		accounts[idx].IsForeign = acct.Get("account.is-foreign").Bool()
		accounts[idx].FundingDate = acct.Get("account.funding-date").Time()
	}

	return accounts, nil
}
