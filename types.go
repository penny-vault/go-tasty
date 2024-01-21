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

package gotasty

import (
	"sync"
	"sync/atomic"
	"time"
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

	Debug bool // print details of each response and request

	RefreshLocker *sync.Mutex
}

// SessionOpts provide additional settings when creating a new tastytrade Open API session
type SessionOpts struct {
	// request a remember-me token which enables the API to refresh session
	// tokens for up-to 28 days
	RememberMe bool

	// use the tastytrade Open API sandbox environment for testing
	Sandbox bool

	// enable debug mode which prints the status of each request
	Debug bool
}

// User is used to authenticate a user session
type User struct {
	Username      string `json:"login"`
	Password      string `json:"password,omitempty"`
	RememberMe    bool   `json:"remember-me"`
	RememberToken string `json:"remember-token,omitempty"`
}

// AccountInfo stores information about the accounts available to the current customer
type AccountInfo struct {
	AccountNumber     string    `json:"account-number"`    // account number, e.g. 5WT0001
	ExternalID        string    `json:"external-id"`       // external identifier, e.g. A0000196557
	OpenedAt          time.Time `json:"opened-at"`         // time the account was opened
	Nickname          string    `json:"nickname"`          // customer assigned nickname for account
	AccountType       string    `json:"account-type-name"` // type of account
	DayTraderStatus   bool      `json:"day-trader-status"` // if account is flagged as a pattern day trader
	IsFirmError       bool      `json:"is-firm-error"`
	IsFirmProprietary bool      `json:"is-firm-proprietary"`
	IsTestDrive       bool      `json:"is-test-drive"`
	MarginOrCash      string    `json:"margin-or-cash"`
	IsForeign         bool      `json:"is-foreign"`
	FundingDate       time.Time `json:"funding-date"`
	AuthorityLevel    string    `json:"authority-level"`
}

/*
{
	"has-institutional-assets": "string",
	"visa-expiration-date": "string",
	"gender": "string",
	"second-surname": "string",
	"last-name": "string",
	"political-organization": "string",
	"middle-name": "string",
	"entity": {
	  "is-domestic": "string",
	  "entity-type": "string",
	  "entity-officers": [
		{
		  "relationship-to-entity": "string",
		  "visa-expiration-date": "2024-01-20",
		  "last-name": "string",
		  "middle-name": "string",
		  "work-phone-number": "string",
		  "prefix-name": "string",
		  "visa-type": "string",
		  "number-of-dependents": "string",
		  "suffix-name": "string",
		  "job-title": "string",
		  "birth-country": "string",
		  "first-name": "string",
		  "occupation": "string",
		  "marital-status": "string",
		  "tax-number": "string",
		  "citizenship-country": "string",
		  "usa-citizenship-type": "string",
		  "owner-of-record": true,
		  "is-foreign": "string",
		  "employment-status": "string",
		  "mobile-phone-number": "string",
		  "address": {
			"is-domestic": "string",
			"street-two": "string",
			"city": "string",
			"postal-code": "string",
			"state-region": "string",
			"is-foreign": "string",
			"street-three": "string",
			"country": "string",
			"street-one": "string"
		  },
		  "home-phone-number": "string",
		  "id": "string",
		  "tax-number-type": "string",
		  "email": "string",
		  "birth-date": "2024-01-20",
		  "employer-name": "string",
		  "external-id": "string"
		}
	  ],
	  "phone-number": "string",
	  "grantor-birth-date": "string",
	  "has-foreign-bank-affiliation": "string",
	  "tax-number": "string",
	  "grantor-email": "string",
	  "has-foreign-institution-affiliation": "string",
	  "entity-suitability": {
		"entity-id": 0,
		"tax-bracket": "string",
		"annual-net-income": 0,
		"liquid-net-worth": 0,
		"stock-trading-experience": "string",
		"futures-trading-experience": "string",
		"uncovered-options-trading-experience": "string",
		"id": "string",
		"covered-options-trading-experience": "string",
		"net-worth": 0
	  },
	  "grantor-last-name": "string",
	  "business-nature": "string",
	  "address": {
		"is-domestic": "string",
		"street-two": "string",
		"city": "string",
		"postal-code": "string",
		"state-region": "string",
		"is-foreign": "string",
		"street-three": "string",
		"country": "string",
		"street-one": "string"
	  },
	  "grantor-middle-name": "string",
	  "grantor-first-name": "string",
	  "grantor-tax-number": "string",
	  "id": "string",
	  "email": "string",
	  "legal-name": "string",
	  "foreign-institution": "string"
	},
	"work-phone-number": "string",
	"permitted-account-types": "string",
	"foreign-tax-number": "string",
	"listed-affiliation-symbol": "string",
	"prefix-name": "string",
	"visa-type": "string",
	"suffix-name": "string",
	"birth-country": "string",
	"first-name": "string",
	"signature-of-agreement": true,
	"is-professional": true,
	"agreed-to-terms": true,
	"industry-affiliation-firm": "string",
	"subject-to-tax-withholding": true,
	"tax-number": "string",
	"citizenship-country": "string",
	"usa-citizenship-type": "string",
	"has-political-affiliation": true,
	"customer-suitability": {
	  "tax-bracket": "string",
	  "number-of-dependents": 0,
	  "annual-net-income": 0,
	  "job-title": "string",
	  "customer-id": 0,
	  "occupation": "string",
	  "marital-status": "string",
	  "liquid-net-worth": 0,
	  "stock-trading-experience": "string",
	  "employment-status": "string",
	  "futures-trading-experience": "string",
	  "uncovered-options-trading-experience": "string",
	  "id": "string",
	  "covered-options-trading-experience": "string",
	  "employer-name": "string",
	  "net-worth": 0
	},
	"identifiable-type": "string",
	"is-foreign": "string",
	"mailing-address": {
	  "is-domestic": "string",
	  "street-two": "string",
	  "city": "string",
	  "postal-code": "string",
	  "state-region": "string",
	  "is-foreign": "string",
	  "street-three": "string",
	  "country": "string",
	  "street-one": "string"
	},
	"has-listed-affiliation": true,
	"is-investment-adviser": "string",
	"mobile-phone-number": "string",
	"has-industry-affiliation": true,
	"address": {
	  "is-domestic": "string",
	  "street-two": "string",
	  "city": "string",
	  "postal-code": "string",
	  "state-region": "string",
	  "is-foreign": "string",
	  "street-three": "string",
	  "country": "string",
	  "street-one": "string"
	},
	"person": {
	  "visa-expiration-date": "2024-01-20",
	  "last-name": "string",
	  "middle-name": "string",
	  "prefix-name": "string",
	  "visa-type": "string",
	  "number-of-dependents": "string",
	  "suffix-name": "string",
	  "job-title": "string",
	  "birth-country": "string",
	  "first-name": "string",
	  "occupation": "string",
	  "marital-status": "string",
	  "citizenship-country": "string",
	  "usa-citizenship-type": "string",
	  "employment-status": "string",
	  "birth-date": "2024-01-20",
	  "employer-name": "string",
	  "external-id": "string"
	},
	"has-delayed-quotes": true,
	"desk-customer-id": "string",
	"home-phone-number": "string",
	"agreed-to-margining": true,
	"id": "string",
	"has-pending-or-approved-application": "string",
	"tax-number-type": "string",
	"email": "string",
	"birth-date": "string",
	"user-id": "string",
	"external-id": "string",
	"family-member-names": "string",
	"first-surname": "string"
  }
*/
