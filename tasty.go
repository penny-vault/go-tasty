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
	"net/url"
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
	defer uncompress.Close()

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

	if err := compressor.Close(); err != nil {
		return []byte{}, err
	}

	return out.Bytes(), nil
}

// Delete invalidates the session token and remember token so they may no-longer be used
func (session *Session) Delete() error {
	client, err := session.restyClient()
	if err != nil {
		return err
	}

	resp, err := client.R().Delete("/sessions")
	if err != nil {
		return err
	}

	if resp.StatusCode() >= 400 {
		return fmt.Errorf("%s: %s", resp.Status(), resp.Body())
	}

	return nil
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
func (session *Session) Accounts() ([]*Account, error) {
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
	accounts := make([]*Account, len(arr))
	for idx, acct := range arr {
		accounts[idx] = &Account{
			AccountNumber:     acct.Get("account.account-number").String(),
			ExternalID:        acct.Get("account.external-id").String(),
			OpenedAt:          acct.Get("account.opened-at").Time(),
			Nickname:          acct.Get("account.nickname").String(),
			AccountType:       acct.Get("account.account-type-name").String(),
			DayTraderStatus:   acct.Get("account.day-trader-status").Bool(),
			MarginOrCash:      acct.Get("account.margin-or-cash").String(),
			AuthorityLevel:    acct.Get("authority-level").String(),
			IsFirmError:       acct.Get("account.is-firm-error").Bool(),
			IsFirmProprietary: acct.Get("account.is-firm-proprietary").Bool(),
			IsTestDrive:       acct.Get("account.is-test-drive").Bool(),
			IsForeign:         acct.Get("account.is-foreign").Bool(),
			FundingDate:       acct.Get("account.funding-date").Time(),
		}
	}

	return accounts, nil
}

// Balance returns the current balance values for an account
func (session *Session) Balance(accountNumber string) (*Balance, error) {
	client, err := session.restyClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.R().Get(fmt.Sprintf("/accounts/%s/balances", accountNumber))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("%s (balances): %s", resp.Status(), resp.Body())
	}

	body := string(resp.Body())

	balance := &Balance{
		AccountNumber:                      gjson.Get(body, "data.account-number").String(),
		CashBalance:                        gjson.Get(body, "data.cash-balance").Float(),
		LongEquityValue:                    gjson.Get(body, "data.long-equity-value").Float(),
		ShortEquityValue:                   gjson.Get(body, "data.short-equity-value").Float(),
		LongDerivativeValue:                gjson.Get(body, "data.long-derivative-value").Float(),
		ShortDerivativeValue:               gjson.Get(body, "data.short-derivative-value").Float(),
		LongFuturesValue:                   gjson.Get(body, "data.long-futures-value").Float(),
		ShortFuturesValue:                  gjson.Get(body, "data.short-futures-value").Float(),
		LongFuturesDerivativeValue:         gjson.Get(body, "data.long-futures-derivative-value").Float(),
		ShortFuturesDerivativeValue:        gjson.Get(body, "data.short-futures-derivative-value").Float(),
		LongMargineableValue:               gjson.Get(body, "data.long-margineable-value").Float(),
		ShortMargineableValue:              gjson.Get(body, "data.short-margineable-value").Float(),
		MarginEquity:                       gjson.Get(body, "data.margin-equity").Float(),
		EquityBuyingPower:                  gjson.Get(body, "data.equity-buying-power").Float(),
		DerivativeBuyingPower:              gjson.Get(body, "data.derivative-buying-power").Float(),
		DayTradingBuyingPower:              gjson.Get(body, "data.day-trading-buying-power").Float(),
		FuturesMarginRequirement:           gjson.Get(body, "data.futures-margin-requirement").Float(),
		AvailableTradingFunds:              gjson.Get(body, "data.available-trading-funds").Float(),
		MaintenanceRequirement:             gjson.Get(body, "data.maintenance-requirement").Float(),
		MaintenanceCallValue:               gjson.Get(body, "data.maintenance-call-value").Float(),
		RegTCallValue:                      gjson.Get(body, "data.reg-t-call-value").Float(),
		DayTradingCallValue:                gjson.Get(body, "data.day-trading-call-value").Float(),
		DayEquityCallValue:                 gjson.Get(body, "data.day-equity-call-value").Float(),
		NetLiquidatingValue:                gjson.Get(body, "data.net-liquidating-value").Float(),
		CashAvailableToWithdraw:            gjson.Get(body, "data.cash-available-to-withdraw").Float(),
		DayTradeExcess:                     gjson.Get(body, "data.day-trade-excess").Float(),
		PendingCash:                        gjson.Get(body, "data.pending-cash").Float(),
		PendingCashEffect:                  gjson.Get(body, "data.pending-cash-effect").String(),
		LongCryptocurrencyValue:            gjson.Get(body, "data.long-cryptocurrency-value").Float(),
		ShortCryptocurrencyValue:           gjson.Get(body, "data.short-cryptocurrency-value").Float(),
		CryptocurrencyMarginRequirement:    gjson.Get(body, "data.cryptocurrency-margin-requirement").Float(),
		UnsettledCryptocurrencyFiatAmount:  gjson.Get(body, "data.unsettled-cryptocurrency-fiat-amount").Float(),
		UnsettledCryptocurrencyFiatEffect:  gjson.Get(body, "data.unsettled-cryptocurrency-fiat-effect").String(),
		ClosedLoopAvailableBalance:         gjson.Get(body, "data.closed-loop-available-balance").Float(),
		EquityOfferingMarginRequirement:    gjson.Get(body, "data.equity-offering-margin-requirement").Float(),
		LongBondValue:                      gjson.Get(body, "data.long-bond-value").Float(),
		BondMarginRequirement:              gjson.Get(body, "data.bond-margin-requirement").Float(),
		UsedDerivativeBuyingPower:          gjson.Get(body, "data.used-derivative-buying-power").Float(),
		SnapshotDate:                       gjson.Get(body, "data.snapshot-date").Time(),
		RegTMarginRequirement:              gjson.Get(body, "data.reg-t-margin-requirement").Float(),
		FuturesOvernightMarginRequirement:  gjson.Get(body, "data.futures-overnight-margin-requirement").Float(),
		FuturesIntradayMarginRequirement:   gjson.Get(body, "data.futures-intraday-margin-requirement").Float(),
		MaintenanceExcess:                  gjson.Get(body, "data.maintenance-excess").Float(),
		PendingMarginInterest:              gjson.Get(body, "data.pending-margin-interest").Float(),
		EffectiveCryptocurrencyBuyingPower: gjson.Get(body, "data.effective-cryptocurrency-buying-power").Float(),
		UpdatedAt:                          gjson.Get(body, "data.updated-at").Time(),
	}

	return balance, nil
}

// BalanceSnapshot returns a snapshot of the account balance at the specified time
func (session *Session) BalanceSnapshot(accountNumber string, timeOfDay TimeOfDay, snapshotDate time.Time) (*Balance, error) {
	client, err := session.restyClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.R().Get(fmt.Sprintf("/accounts/%s/balance-snapshots", accountNumber))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("%s (balance-snapshots): %s", resp.Status(), resp.Body())
	}

	body := string(resp.Body())

	balance := &Balance{
		AccountNumber:                      gjson.Get(body, "data.account-number").String(),
		CashBalance:                        gjson.Get(body, "data.cash-balance").Float(),
		LongEquityValue:                    gjson.Get(body, "data.long-equity-value").Float(),
		ShortEquityValue:                   gjson.Get(body, "data.short-equity-value").Float(),
		LongDerivativeValue:                gjson.Get(body, "data.long-derivative-value").Float(),
		ShortDerivativeValue:               gjson.Get(body, "data.short-derivative-value").Float(),
		LongFuturesValue:                   gjson.Get(body, "data.long-futures-value").Float(),
		ShortFuturesValue:                  gjson.Get(body, "data.short-futures-value").Float(),
		LongFuturesDerivativeValue:         gjson.Get(body, "data.long-futures-derivative-value").Float(),
		ShortFuturesDerivativeValue:        gjson.Get(body, "data.short-futures-derivative-value").Float(),
		LongMargineableValue:               gjson.Get(body, "data.long-margineable-value").Float(),
		ShortMargineableValue:              gjson.Get(body, "data.short-margineable-value").Float(),
		MarginEquity:                       gjson.Get(body, "data.margin-equity").Float(),
		EquityBuyingPower:                  gjson.Get(body, "data.equity-buying-power").Float(),
		DerivativeBuyingPower:              gjson.Get(body, "data.derivative-buying-power").Float(),
		DayTradingBuyingPower:              gjson.Get(body, "data.day-trading-buying-power").Float(),
		FuturesMarginRequirement:           gjson.Get(body, "data.futures-margin-requirement").Float(),
		AvailableTradingFunds:              gjson.Get(body, "data.available-trading-funds").Float(),
		MaintenanceRequirement:             gjson.Get(body, "data.maintenance-requirement").Float(),
		MaintenanceCallValue:               gjson.Get(body, "data.maintenance-call-value").Float(),
		RegTCallValue:                      gjson.Get(body, "data.reg-t-call-value").Float(),
		DayTradingCallValue:                gjson.Get(body, "data.day-trading-call-value").Float(),
		DayEquityCallValue:                 gjson.Get(body, "data.day-equity-call-value").Float(),
		NetLiquidatingValue:                gjson.Get(body, "data.net-liquidating-value").Float(),
		CashAvailableToWithdraw:            gjson.Get(body, "data.cash-available-to-withdraw").Float(),
		DayTradeExcess:                     gjson.Get(body, "data.day-trade-excess").Float(),
		PendingCash:                        gjson.Get(body, "data.pending-cash").Float(),
		PendingCashEffect:                  gjson.Get(body, "data.pending-cash-effect").String(),
		LongCryptocurrencyValue:            gjson.Get(body, "data.long-cryptocurrency-value").Float(),
		ShortCryptocurrencyValue:           gjson.Get(body, "data.short-cryptocurrency-value").Float(),
		CryptocurrencyMarginRequirement:    gjson.Get(body, "data.cryptocurrency-margin-requirement").Float(),
		UnsettledCryptocurrencyFiatAmount:  gjson.Get(body, "data.unsettled-cryptocurrency-fiat-amount").Float(),
		UnsettledCryptocurrencyFiatEffect:  gjson.Get(body, "data.unsettled-cryptocurrency-fiat-effect").String(),
		ClosedLoopAvailableBalance:         gjson.Get(body, "data.closed-loop-available-balance").Float(),
		EquityOfferingMarginRequirement:    gjson.Get(body, "data.equity-offering-margin-requirement").Float(),
		LongBondValue:                      gjson.Get(body, "data.long-bond-value").Float(),
		BondMarginRequirement:              gjson.Get(body, "data.bond-margin-requirement").Float(),
		UsedDerivativeBuyingPower:          gjson.Get(body, "data.used-derivative-buying-power").Float(),
		SnapshotDate:                       gjson.Get(body, "data.snapshot-date").Time(),
		RegTMarginRequirement:              gjson.Get(body, "data.reg-t-margin-requirement").Float(),
		FuturesOvernightMarginRequirement:  gjson.Get(body, "data.futures-overnight-margin-requirement").Float(),
		FuturesIntradayMarginRequirement:   gjson.Get(body, "data.futures-intraday-margin-requirement").Float(),
		MaintenanceExcess:                  gjson.Get(body, "data.maintenance-excess").Float(),
		PendingMarginInterest:              gjson.Get(body, "data.pending-margin-interest").Float(),
		EffectiveCryptocurrencyBuyingPower: gjson.Get(body, "data.effective-cryptocurrency-buying-power").Float(),
		UpdatedAt:                          gjson.Get(body, "data.updated-at").Time(),
	}

	return balance, nil
}

// Positions returns a list of the accounts positions
func (session *Session) Positions(accountNumber string, filterOpts ...PositionFilterOpts) ([]*Position, error) {
	client, err := session.restyClient()
	if err != nil {
		return nil, err
	}

	req := client.R()

	// set parameters from filterOpts
	if len(filterOpts) > 1 {
		filter := filterOpts[0]

		if len(filter.UnderlyingSymbol) > 0 {
			req = req.SetQueryParamsFromValues(url.Values{
				"underlying-symbol[]": filter.UnderlyingSymbol,
			})
		}

		if filter.Symbol != "" {
			req = req.SetQueryParam("symbol", filter.Symbol)
		}

		if filter.InstrumentType != UndefinedInstrument {
			req = req.SetQueryParam("instrument-type", filter.InstrumentType.String())
		}

		if filter.IncludeClosedPositions {
			req = req.SetQueryParam("include-closed-positions", "true")
		}

		if filter.UnderlyingProductCode != "" {
			req = req.SetQueryParam("underlying-product-code", filter.UnderlyingProductCode)
		}

		if len(filter.PartitionKeys) > 0 {
			req = req.SetQueryParamsFromValues(url.Values{
				"partition-keys[]": filter.PartitionKeys,
			})
		}

		if filter.NetPositions {
			req = req.SetQueryParam("net-positions", "true")
		}

		if filter.IncludeMarks {
			req = req.SetQueryParam("include-marks", "true")
		}

	}

	resp, err := req.Get(fmt.Sprintf("/accounts/%s/positions", accountNumber))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("%s (positions): %s", resp.Status(), resp.Body())
	}

	arr := gjson.Get(string(resp.Body()), "data.items").Array()
	positions := make([]*Position, len(arr))
	for idx, pos := range arr {
		positions[idx] = &Position{
			AccountNumber:                 pos.Get("account-number").String(),
			Symbol:                        pos.Get("symbol").String(),
			InstrumentType:                pos.Get("instrument-type").String(),
			UnderlyingSymbol:              pos.Get("underlying-symbol").String(),
			Quantity:                      pos.Get("quantity").Float(),
			QuantityDirection:             pos.Get("quantity-direction").String(),
			ClosePrice:                    pos.Get("close-price").Float(),
			AverageOpenPrice:              pos.Get("average-open-price").Float(),
			AverageYearlyMarketClosePrice: pos.Get("average-yearly-market-close-price").Float(),
			AverageDailyMarketClosePrice:  pos.Get("average-daily-market-close-price").Float(),
			Multiplier:                    pos.Get("multiplier").Float(),
			CostEffect:                    pos.Get("cost-effect").String(),
			IsSuppressed:                  pos.Get("is-suppressed").Bool(),
			IsFrozen:                      pos.Get("is-frozen").Bool(),
			RestrictedQuantity:            pos.Get("restricted-quantity").Float(),
			RealizedDayGain:               pos.Get("realized-day-gain").Float(),
			RealizedDayGainEffect:         pos.Get("realized-day-gain-effect").String(),
			RealizedDayGainDate:           pos.Get("realized-day-gain-date").Time(),
			RealizedToday:                 pos.Get("realized-today").Float(),
			RealizedTodayEffect:           pos.Get("realized-today-effect").String(),
			RealizedTodayDate:             pos.Get("realized-today-date").Time(),
			ExpiresAt:                     pos.Get("expires-at").Time(),
			CreatedAt:                     pos.Get("created-at").Time(),
			UpdatedAt:                     pos.Get("updated-at").Time(),
		}
	}

	return positions, nil
}

// Transactions returns a list of the accounts transactions
func (session *Session) Transactions(accountNumber string, filterOpts ...TransactionFilterOpts) ([]*Transaction, error) {
	client, err := session.restyClient()
	if err != nil {
		return nil, err
	}

	req := client.R()

	// set parameters from filterOpts
	if len(filterOpts) > 1 {
		filter := filterOpts[0]

		if filter.PerPage > 0 {
			req = req.SetQueryParam("per-page", fmt.Sprint(filter.PerPage))
		}

		if filter.PageOffset > 0 {
			req = req.SetQueryParam("page-offset", fmt.Sprint(filter.PageOffset))
		}

		req = req.SetQueryParam("sort", filter.Sort.String())

		if len(filter.TransactionTypes) == 1 {
			req = req.SetQueryParam("type", filter.TransactionTypes[0])
		} else if len(filter.TransactionTypes) > 1 {
			req = req.SetQueryParamsFromValues(url.Values{
				"types[]": filter.TransactionTypes,
			})
		}

		if len(filter.TransactionSubTypes) > 0 {
			req = req.SetQueryParamsFromValues(url.Values{
				"sub-type[]": filter.TransactionSubTypes,
			})
		}

		if filter.StartDate.After(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) {
			req = req.SetQueryParam("start-date", filter.StartDate.Format(time.RFC3339))
		}

		if filter.EndDate.After(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) {
			req = req.SetQueryParam("end-date", filter.EndDate.Format(time.RFC3339))
		}

		if filter.Symbol != "" {
			req = req.SetQueryParam("symbol", filter.Symbol)
		}

		if filter.InstrumentType != UndefinedInstrument {
			req = req.SetQueryParam("instrument-type", filter.InstrumentType.String())
		}

		if filter.UnderlyingSymbol != "" {
			req = req.SetQueryParam("underlying-symbol", filter.UnderlyingSymbol)
		}

		if filter.Action != UndefinedAction {
			req = req.SetQueryParam("action", filter.Action.String())
		}

		if filter.PartitionKey != "" {
			req = req.SetQueryParam("partition-key", filter.PartitionKey)
		}

		if filter.FuturesSymbol != "" {
			req = req.SetQueryParam("futures-symbol", filter.FuturesSymbol)
		}
	}

	resp, err := req.Get(fmt.Sprintf("/accounts/%s/transactions", accountNumber))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("%s (transactions): %s", resp.Status(), resp.Body())
	}

	arr := gjson.Get(string(resp.Body()), "data.items").Array()
	transactions := make([]*Transaction, len(arr))
	for idx, trx := range arr {
		instrumentType := InstrumentTypeFromString(trx.Get("instrument-type").String())
		actionType := ActionTypeFromString(trx.Get("action").String())
		valueEffect := EffectFromString(trx.Get("value-effect").String())
		regulatoryFeesEffect := EffectFromString(trx.Get("regulatory-fees-effect").String())
		clearingFeesEffect := EffectFromString(trx.Get("clearing-fees-effect").String())
		otherChargeEffect := EffectFromString(trx.Get("other-charge-effect").String())
		netValueEffect := EffectFromString(trx.Get("net-value-effect").String())
		commissionEffect := EffectFromString(trx.Get("commission-effect").String())
		proprietaryIndexOptionFeesEffect := EffectFromString(trx.Get("proprietary-index-option-fees-effect").String())

		lotArr := trx.Get("lots").Array()
		lots := make([]*Lot, len(lotArr))
		for idx2, lot := range lotArr {
			lots[idx2] = &Lot{
				ID:                lot.Get("id").String(),
				TransactionID:     lot.Get("transaction-id").Int(),
				Quantity:          lot.Get("quantity").Float(),
				Price:             lot.Get("price").Float(),
				QuantityDirection: lot.Get("quantity-direction").String(),
				ExecutedAt:        lot.Get("executed-at").Time(),
				TransactionDate:   lot.Get("transaction-date").Time(),
			}
		}

		transactions[idx] = &Transaction{
			ID:                               trx.Get("id").Int(),
			AccountNumber:                    trx.Get("account-number").String(),
			ExecutedAt:                       trx.Get("executed-at").Time(),
			TransactionDate:                  trx.Get("transaction-date").Time(),
			TransactionType:                  trx.Get("transaction-type").String(),
			TransactionSubType:               trx.Get("transaction-sub-type").String(),
			Description:                      trx.Get("description").String(),
			UnderlyingSymbol:                 trx.Get("underlying-symbol").String(),
			InstrumentType:                   instrumentType,
			Symbol:                           trx.Get("symbol").String(),
			Action:                           actionType,
			Quantity:                         trx.Get("quantity").Float(),
			Price:                            trx.Get("price").Float(),
			Value:                            trx.Get("value").Float(),
			ValueEffect:                      valueEffect,
			RegulatoryFees:                   trx.Get("regulatory-fees").Float(),
			RegulatoryFeesEffect:             regulatoryFeesEffect,
			ClearingFees:                     trx.Get("clearing-fees").Float(),
			ClearingFeesEffect:               clearingFeesEffect,
			OtherCharge:                      trx.Get("other-charge").Float(),
			OtherChargeEffect:                otherChargeEffect,
			OtherChargeDescription:           trx.Get("other-charge-description").String(),
			NetValue:                         trx.Get("net-value").Float(),
			NetValueEffect:                   netValueEffect,
			Commission:                       trx.Get("commission").Float(),
			CommissionEffect:                 commissionEffect,
			ProprietaryIndexOptionFees:       trx.Get("proprietary-index-option-fees").Float(),
			ProprietaryIndexOptionFeesEffect: proprietaryIndexOptionFeesEffect,
			IsEstimatedFee:                   trx.Get("is-estimated-fee").Bool(),
			OrderID:                          trx.Get("order-id").Int(),
			Lots:                             lots,
			LegCount:                         trx.Get("leg-count").Int(),
			DestinationVenue:                 trx.Get("destination-venue").String(),
			AgencyPrice:                      trx.Get("agency-price").Float(),
			PrincipalPrice:                   trx.Get("principal-price").Float(),
			ExternalExchangeOrderNumber:      trx.Get("ext-exchange-order-number").String(),
			ExternalGlobalOrderNumber:        trx.Get("ext-global-order-number").Int(),
			ExternalGroupID:                  trx.Get("ext-group-id").String(),
			ExternalGroupFillID:              trx.Get("ext-group-fill-id").String(),
			ExternalExecutionID:              trx.Get("ext-exec-id").String(),
			ExecutionID:                      trx.Get("exec-id").String(),
			Exchange:                         trx.Get("exchange").String(),
			ReversesID:                       trx.Get("reverses-id").Int(),
			ExchangeAffiliationID:            trx.Get("exchange-affiliation-identifier").String(),
			CostBasisReconciliationDate:      trx.Get("cost-basis-reconciliation-date").Time(),
		}
	}

	return transactions, nil
}

// Orders returns a paginated list of the accounts's orders
func (session *Session) Orders(accountNumber string, filterOpts ...OrdersFilterOpts) ([]*OrderStatus, error) {
	client, err := session.restyClient()
	if err != nil {
		return nil, err
	}

	req := client.R()

	// set parameters from filterOpts
	if len(filterOpts) > 1 {
		filter := filterOpts[0]

		if filter.PerPage > 0 {
			req = req.SetQueryParam("per-page", fmt.Sprint(filter.PerPage))
		}

		if filter.PageOffset > 0 {
			req = req.SetQueryParam("page-offset", fmt.Sprint(filter.PageOffset))
		}

		req = req.SetQueryParam("sort", filter.Sort.String())

		if len(filter.Status) > 0 {
			req = req.SetQueryParamsFromValues(url.Values{
				"status[]": filter.Status,
			})
		}

		if filter.StartDate.After(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) {
			req = req.SetQueryParam("start-date", filter.StartDate.Format(time.RFC3339))
		}

		if filter.EndDate.After(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) {
			req = req.SetQueryParam("end-date", filter.EndDate.Format(time.RFC3339))
		}

		if filter.UnderlyingSymbol != "" {
			req = req.SetQueryParam("underlying-symbol", filter.UnderlyingSymbol)
		}

		if filter.UnderlyingInstrumentType != UndefinedInstrument {
			req = req.SetQueryParam("underlying-instrument-type", filter.UnderlyingInstrumentType.String())
		}

		if filter.FuturesSymbol != "" {
			req = req.SetQueryParam("futures-symbol", filter.FuturesSymbol)
		}
	}

	resp, err := req.Get(fmt.Sprintf("/accounts/%s/orders", accountNumber))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("%s (orders): %s", resp.Status(), resp.Body())
	}

	arr := gjson.Get(string(resp.Body()), "data.items").Array()
	orders := make([]*OrderStatus, len(arr))
	for idx, order := range arr {
		orders[idx] = parseOrderStatus(order)
	}

	return orders, nil
}

// SubmitOrder sends the specified order to tastytrade for execution
func (session *Session) SubmitOrder(accountNumber string, order *Order) (*OrderResponse, error) {
	client, err := session.restyClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.R().
		SetBody(order).
		Post(fmt.Sprintf("/sessions/%s/orders", accountNumber))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("%s: %s", resp.Status(), resp.Body())
	}

	content := string(resp.Body())
	orderStatus := gjson.Get(content, "data.order")

	return &OrderResponse{
		Order:               parseOrderStatus(orderStatus),
		EffectOnBuyingPower: parseEffectOnBuyingPower(gjson.Get(content, "data.buying-power-effect")),
		FeeCalculation:      parseFeeInfo(gjson.Get(content, "data.fee-calculation")),
		Errors:              parseErrors(gjson.Get(content, "data.errors").Array()),
		Warnings:            parseErrors(gjson.Get(content, "data.warnings").Array()),
	}, nil
}

// DeleteOrder attempts to delete orderID
func (session *Session) DeleteOrder(accountNumber string, orderID string) (*OrderStatus, error) {
	client, err := session.restyClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.R().
		Delete(fmt.Sprintf("/sessions/%s/orders/%s", accountNumber, orderID))
	if err != nil {
		return nil, err
	}

	content := string(resp.Body())
	order := gjson.Get(content, "data.order")
	orderStatus := parseOrderStatus(order)

	return orderStatus, nil
}

func parseOrderStatus(order gjson.Result) *OrderStatus {
	underlyingInstrumentType := InstrumentTypeFromString(order.Get("underlying-instrument-type").String())
	valueEffect := EffectFromString(order.Get("value-effect").String())
	priceEffect := EffectFromString(order.Get("price-effect").String())
	orderType := OrderTypeFromString(order.Get("order-type").String())

	legArr := order.Get("legs").Array()
	legs := make([]*LegStatus, len(legArr))
	for idx2, leg := range legArr {
		instrumentType := InstrumentTypeFromString(leg.Get("instrument-type").String())
		actionType := ActionTypeFromString(leg.Get("action").String())

		fillsArr := leg.Get("fills").Array()
		fills := make([]*FillStatus, len(fillsArr))
		for idx3, fill := range fillsArr {
			fills[idx3] = &FillStatus{
				ExternalGroupFillID: fill.Get("ext-group-fill-id").String(),
				ExternalExecutionID: fill.Get("ext-exec-id").String(),
				FillID:              fill.Get("fill-id").String(),
				Quantity:            fill.Get("quantity").String(),
				FillPrice:           fill.Get("fill-price").Float(),
				FilledAt:            fill.Get("filled-at").Time(),
				DestinationVenue:    fill.Get("destination-venue").String(),
			}
		}

		legs[idx2] = &LegStatus{
			InstrumentType:    instrumentType,
			Symbol:            leg.Get("symbol").String(),
			Quantity:          leg.Get("quantity").String(),
			RemainingQuantity: leg.Get("remaining-quantity").String(),
			Action:            actionType,
			Fills:             fills,
		}
	}

	ruleArr := order.Get("order-rule").Array()
	rules := make([]*RuleStatus, len(ruleArr))
	for idx2, rule := range ruleArr {
		conditionArr := rule.Get("conditions").Array()
		conditions := make([]*ConditionStatus, len(conditionArr))
		for idx3, condition := range conditionArr {
			actionCondition := ActionConditionFromString(condition.Get("action").String())
			instrumentType := InstrumentTypeFromString(condition.Get("instrument-type").String())
			indicatorType := IndicatorFromString(condition.Get("indicator").String())
			comparator := ComparatorFromString(condition.Get("comparator").String())

			priceArr := condition.Get("price-components").Array()
			priceComponents := make([]*ConditionPriceComponents, len(priceArr))
			for idx4, priceComp := range priceArr {
				priceCompInstrument := InstrumentTypeFromString(priceComp.Get("instrument-type").String())

				priceComponents[idx4] = &ConditionPriceComponents{
					Symbol:            priceComp.Get("symbol").String(),
					InstrumentType:    priceCompInstrument,
					Quantity:          priceComp.Get("quantity").String(),
					QuantityDirection: priceComp.Get("quantity-direction").String(),
				}
			}

			conditions[idx3] = &ConditionStatus{
				ID:                         condition.Get("id").String(),
				Action:                     actionCondition,
				TriggeredAt:                condition.Get("triggered-at").Time(),
				TriggeredValue:             condition.Get("triggered-value").Float(),
				Symbol:                     condition.Get("symbol").String(),
				InstrumentType:             instrumentType,
				Indicator:                  indicatorType,
				Comparator:                 comparator,
				Threshold:                  condition.Get("threshold").Float(),
				IsThresholdBasedOnNotional: condition.Get("is-threshold-based-on-notional").Bool(),
				PriceComponents:            priceComponents,
			}
		}

		rules[idx2] = &RuleStatus{
			RouteAfter:  rule.Get("route-after").Time(),
			RoutedAt:    rule.Get("routed-at").Time(),
			CancelAt:    rule.Get("cancel-at").Time(),
			CancelledAt: rule.Get("cancelled-at").Time(),
			Conditions:  conditions,
		}
	}

	orderStatus := &OrderStatus{
		Size:                     order.Get("size").String(),
		TimeInForce:              order.Get("time-in-force").String(),
		TerminalAt:               order.Get("terminal-at").Time(),
		Editable:                 order.Get("editable").Bool(),
		ContingentStatus:         order.Get("contingent-status").String(),
		Legs:                     legs,
		GTCDate:                  order.Get("gtc-date").Time(),
		UpdatedAt:                order.Get("updated-at").String(),
		InFlightAt:               order.Get("in-flight-at").Time(),
		ReplacesOrderID:          order.Get("replaces-order-id").String(),
		UnderlyingSymbol:         order.Get("underlying-symbol").String(),
		Edited:                   order.Get("edited").Bool(),
		Price:                    order.Get("price").Float(),
		CancelUsername:           order.Get("cancel-username").String(),
		AccountNumber:            order.Get("account-number").String(),
		ConfirmationStatus:       order.Get("confirmation-status").String(),
		CancelUserID:             order.Get("cancel-user-id").String(),
		Cancellable:              order.Get("cancellable").Bool(),
		ValueEffect:              valueEffect,
		StopTrigger:              order.Get("stop-trigger").String(),
		CancelledAt:              order.Get("cancelled-at").Time(),
		UnderlyingInstrumentType: underlyingInstrumentType,
		Value:                    order.Get("value").Float(),
		RejectReason:             order.Get("reject-reason").String(),
		Status:                   order.Get("status").String(),
		LiveAt:                   order.Get("live-at").Time(),
		PreflightID:              order.Get("preflight-id").String(),
		PriceEffect:              priceEffect,
		Username:                 order.Get("username").String(),
		ReplacingOrderID:         order.Get("replacing-order-id").String(),
		ComplexOrderID:           order.Get("complex-order-id").String(),
		OrderType:                orderType,
		ID:                       order.Get("id").String(),
		OrderRule:                rules,
		UserId:                   order.Get("user-id").String(),
		ComplexOrderTag:          order.Get("complex-order-tag").String(),
		ReceivedAt:               order.Get("received-at").Time(),
	}

	return orderStatus
}

func parseEffectOnBuyingPower(result gjson.Result) *BuyingPowerChange {
	return &BuyingPowerChange{
		ChangeInMarginRequirement:            result.Get("change-in-margin-requirement").Float(),
		ChangeInMarginRequirementEffect:      EffectFromString(result.Get("change-in-margin-requirement-effect").String()),
		ChangeInBuyingPower:                  result.Get("change-in-buying-power").Float(),
		ChangeInBuyingPowerEffect:            EffectFromString(result.Get("change-in-buying-power-effect").String()),
		CurrentBuyingPower:                   result.Get("current-buying-power").Float(),
		CurrentBuyingPowerEffect:             EffectFromString(result.Get("current-buying-power-effect").String()),
		NewBuyingPower:                       result.Get("new-buying-power").Float(),
		NewBuyingPowerEffect:                 EffectFromString(result.Get("new-buying-power-effect").String()),
		IsolatedOrderMarginRequirement:       result.Get("isolated-order-margin-requirement").Float(),
		IsolatedOrderMarginRequirementEffect: EffectFromString(result.Get("isolated-order-margin-requirement-effect").String()),
		IsSpread:                             result.Get("is-spread").Bool(),
		Impact:                               result.Get("impact").Float(),
		EffectOnCash:                         EffectFromString(result.Get("effect").String()),
	}
}

func parseFeeInfo(result gjson.Result) *FeeInfo {
	return &FeeInfo{
		RegulatoryFees:                   result.Get("regulatory-fees").Float(),
		RegulatoryFeesEffect:             EffectFromString(result.Get("regulatory-fees-effect").String()),
		ClearingFees:                     result.Get("clearing-fees").Float(),
		ClearingFeesEffect:               EffectFromString(result.Get("clearing-fees-effect").String()),
		Commission:                       result.Get("commission").Float(),
		CommissionEffect:                 EffectFromString(result.Get("commission-effect").String()),
		ProprietaryIndexOptionFees:       result.Get("proprietary-index-option-fees").Float(),
		ProprietaryIndexOptionFeesEffect: EffectFromString(result.Get("proprietary-index-option-fees-effect").String()),
		TotalFees:                        result.Get("total-fees").Float(),
		TotalFeesEffect:                  EffectFromString(result.Get("total-fees-effect").String()),
	}
}

func parseErrors(arr []gjson.Result) []*ErrorMsg {
	errorArr := make([]*ErrorMsg, len(arr))
	for idx, errorMsg := range arr {
		errorArr[idx] = &ErrorMsg{
			Code:        errorMsg.Get("code").String(),
			Message:     errorMsg.Get("message").String(),
			PreflightID: errorMsg.Get("preflight-id").String(),
		}
	}
	return errorArr
}
