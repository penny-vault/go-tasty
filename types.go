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

// User is used to authenticate a user session
type User struct {
	Username      string `json:"login"`
	Password      string `json:"password,omitempty"`
	RememberMe    bool   `json:"remember-me"`
	RememberToken string `json:"remember-token,omitempty"`
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
