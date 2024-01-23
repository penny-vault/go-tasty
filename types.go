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

type SortDirection int

const (
	Desc SortDirection = iota
	Asc
)

func (sortDirection SortDirection) String() string {
	switch sortDirection {
	case Desc:
		return "desc"
	case Asc:
		return "asc"
	}

	return "UNK"
}

type TimeOfDay int

const (
	BOD TimeOfDay = iota
	EOD
)

func (timeOfDay TimeOfDay) String() string {
	switch timeOfDay {
	case BOD:
		return "BOD"
	case EOD:
		return "EOD"
	}

	return "UNK"
}

type PositionFilterOpts struct {
	Symbol                string
	InstrumentType        InstrumentTypeChoice
	UnderlyingSymbol      []string
	UnderlyingProductCode string

	PartitionKeys []string

	NetPositions           bool
	IncludeClosedPositions bool
	IncludeMarks           bool
}

type TransactionFilterOpts struct {
	StartDate time.Time
	EndDate   time.Time

	Symbol           string
	InstrumentType   InstrumentTypeChoice
	UnderlyingSymbol string
	FuturesSymbol    string
	Action           ActionType

	PartitionKey string

	TransactionTypes    []string
	TransactionSubTypes []string

	Status []string
	Sort   *SortDirection

	// Pagination settings
	PerPage    int
	PageOffset int
}

type OrdersFilterOpts struct {
	StartDate time.Time
	EndDate   time.Time

	UnderlyingSymbol         string
	UnderlyingInstrumentType InstrumentTypeChoice
	FuturesSymbol            string

	NetPositions           bool
	IncludeClosedPositions bool
	IncludeMarks           bool

	Status []string
	Sort   *SortDirection

	// Pagination settings
	PerPage    int
	PageOffset int
}

// Account stores information about the accounts available to the current customer
type Account struct {
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

// Balance details for a specific account
type Balance struct {
	AccountNumber                      string    `json:"account-number"`
	CashBalance                        float64   `json:"cash-balance"`
	LongEquityValue                    float64   `json:"long-equity-value"`
	ShortEquityValue                   float64   `json:"short-equity-value"`
	LongDerivativeValue                float64   `json:"long-derivative-value"`
	ShortDerivativeValue               float64   `json:"short-derivative-value"`
	LongFuturesValue                   float64   `json:"long-futures-value"`
	ShortFuturesValue                  float64   `json:"short-futures-value"`
	LongFuturesDerivativeValue         float64   `json:"long-futures-derivative-value"`
	ShortFuturesDerivativeValue        float64   `json:"short-futures-derivative-value"`
	LongMargineableValue               float64   `json:"long-margineable-value"`
	ShortMargineableValue              float64   `json:"short-margineable-value"`
	MarginEquity                       float64   `json:"margin-equity"`
	EquityBuyingPower                  float64   `json:"equity-buying-power"`
	DerivativeBuyingPower              float64   `json:"derivative-buying-power"`
	DayTradingBuyingPower              float64   `json:"day-trading-buying-power"`
	FuturesMarginRequirement           float64   `json:"futures-margin-requirement"`
	AvailableTradingFunds              float64   `json:"available-trading-funds"`
	MaintenanceRequirement             float64   `json:"maintenance-requirement"`
	MaintenanceCallValue               float64   `json:"maintenance-call-value"`
	RegTCallValue                      float64   `json:"reg-t-call-value"`
	DayTradingCallValue                float64   `json:"day-trading-call-value"`
	DayEquityCallValue                 float64   `json:"day-equity-call-value"`
	NetLiquidatingValue                float64   `json:"net-liquidating-value"`
	CashAvailableToWithdraw            float64   `json:"cash-available-to-withdraw"`
	DayTradeExcess                     float64   `json:"day-trade-excess"`
	PendingCash                        float64   `json:"pending-cash"`
	PendingCashEffect                  string    `json:"pending-cash-effect"`
	LongCryptocurrencyValue            float64   `json:"long-cryptocurrency-value"`
	ShortCryptocurrencyValue           float64   `json:"short-cryptocurrency-value"`
	CryptocurrencyMarginRequirement    float64   `json:"cryptocurrency-margin-requirement"`
	UnsettledCryptocurrencyFiatAmount  float64   `json:"unsettled-cryptocurrency-fiat-amount"`
	UnsettledCryptocurrencyFiatEffect  string    `json:"unsettled-cryptocurrency-fiat-effect"`
	ClosedLoopAvailableBalance         float64   `json:"closed-loop-available-balance"`
	EquityOfferingMarginRequirement    float64   `json:"equity-offering-margin-requirement"`
	LongBondValue                      float64   `json:"long-bond-value"`
	BondMarginRequirement              float64   `json:"bond-margin-requirement"`
	UsedDerivativeBuyingPower          float64   `json:"used-derivative-buying-power"`
	SnapshotDate                       time.Time `json:"snapshot-date"`
	RegTMarginRequirement              float64   `json:"reg-t-margin-requirement"`
	FuturesOvernightMarginRequirement  float64   `json:"futures-overnight-margin-requirement"`
	FuturesIntradayMarginRequirement   float64   `json:"futures-intraday-margin-requirement"`
	MaintenanceExcess                  float64   `json:"maintenance-excess"`
	PendingMarginInterest              float64   `json:"pending-margin-interest"`
	EffectiveCryptocurrencyBuyingPower float64   `json:"effective-cryptocurrency-buying-power"`
	UpdatedAt                          time.Time `json:"updated-at"`
}

// Position stores details about the positions held in an account
//
// A position with a quantity of 0 is considered closed. These are purged
// overnight.
//
// Equity option positions also include an expires-at timestamp.
//
// For P/L calculations, you should rely on the live quote data as much as
// possible to ensure up-to-date calculations (see Streaming Market Data).
// In profit/loss calculations use price from the DXLink Trade
// market event, or bidPrice & askPrice from the DXLink Quote market event.
type Position struct {
	AccountNumber                 string    `json:"account-number"`
	Symbol                        string    `json:"symbol"`
	InstrumentType                string    `json:"instrument-type"`
	UnderlyingSymbol              string    `json:"underlying-symbol"`
	Quantity                      float64   `json:"quantity"`
	QuantityDirection             string    `json:"quantity-direction"`
	ClosePrice                    float64   `json:"close-price"`
	AverageOpenPrice              float64   `json:"average-open-price"`
	AverageYearlyMarketClosePrice float64   `json:"average-yearly-market-close-price"`
	AverageDailyMarketClosePrice  float64   `json:"average-daily-market-close-price"`
	Multiplier                    float64   `json:"multiplier"`
	CostEffect                    string    `json:"cost-effect"`
	IsSuppressed                  bool      `json:"is-suppressed"`
	IsFrozen                      bool      `json:"is-frozen"`
	RestrictedQuantity            float64   `json:"restricted-quantity"`
	RealizedDayGain               float64   `json:"realized-day-gain"`
	RealizedDayGainEffect         string    `json:"realized-day-gain-effect"`
	RealizedDayGainDate           time.Time `json:"realized-day-gain-date"`
	RealizedToday                 float64   `json:"realized-today"`
	RealizedTodayEffect           string    `json:"realized-today-effect"`
	RealizedTodayDate             time.Time `json:"realized-today-date"`
	ExpiresAt                     time.Time `json:"expires-at"`
	CreatedAt                     time.Time `json:"created-at"`
	UpdatedAt                     time.Time `json:"updated-at"`
}

type TimeInForceChoice int

const (
	// Day orders live until either the order fills or the market closes.
	// If a day order does not get filled by the time the market closes,
	// it transitions to expired.
	Day TimeInForceChoice = iota

	// Good 'til Canceled orders never expire. They will work until they
	// are either filled or the customer cancels them.
	GTC

	// Good 'til Date orders expire on a given date. If you submit a GTD order,
	// you must also include a gtc-date in the JSON (Yes, calling it gtd-date would
	// have made more sense - we apologize).
	GTD

	Ext
	GTCExt
	IOC
)

func (timeInForce TimeInForceChoice) String() string {
	switch timeInForce {
	case Day:
		return "Day"
	case GTC:
		return "GTC"
	case GTD:
		return "GTD"
	case Ext:
		return "Ext"
	case GTCExt:
		return "GTC Ext"
	case IOC:
		return "IOC"
	default:
		return "UNK"
	}
}

type OrderTypeChoice int

const (
	UndefinedOrderType OrderTypeChoice = iota
	Limit
	Market
	MarketableLimit
	Stop
	StopLimit
	NotionalMarket
)

func OrderTypeFromString(input string) OrderTypeChoice {
	switch input {
	case "Limit":
		return Limit
	case "Market":
		return Market
	case "Marketable Limit":
		return MarketableLimit
	case "Stop":
		return Stop
	case "StopLimit":
		return StopLimit
	case "Notional Market":
		return NotionalMarket
	}

	return UndefinedOrderType
}

func (orderType OrderTypeChoice) String() string {
	switch orderType {
	case Limit:
		return "Limit"
	case Market:
		return "Market"
	case MarketableLimit:
		return "Marketable Limit"
	case Stop:
		return "Stop"
	case StopLimit:
		return "StopLimit"
	case NotionalMarket:
		return "Notional Market"
	default:
		return "UNK"
	}
}

type Effect int

const (
	UndefinedEffect Effect = iota
	Credit
	Debit
)

func EffectFromString(input string) Effect {
	switch input {
	case "Credit":
		return Credit
	case "Debit":
		return Debit
	}

	return UndefinedEffect
}

func (effect Effect) String() string {
	switch effect {
	case Credit:
		return "Credit"
	case Debit:
		return "Debit"
	default:
		return "UNK"
	}
}

type InstrumentTypeChoice int

const (
	UndefinedInstrument InstrumentTypeChoice = iota
	Cryptocurrency
	Equity
	EquityOffering
	EquityOption
	Future
	FutureOption
)

func InstrumentTypeFromString(input string) InstrumentTypeChoice {
	switch input {
	case "Cryptocurrency":
		return Cryptocurrency
	case "Equity":
		return Equity
	case "Equity Offering":
		return EquityOffering
	case "Equity Option":
		return EquityOption
	case "Future":
		return Future
	case "Future Option":
		return FutureOption
	}

	return UndefinedInstrument
}

func (instrumentType InstrumentTypeChoice) String() string {
	switch instrumentType {
	case Cryptocurrency:
		return "Cryptocurrency"
	case Equity:
		return "Equity"
	case EquityOffering:
		return "Equity Offering"
	case EquityOption:
		return "Equity Option"
	case Future:
		return "Future"
	case FutureOption:
		return "Future Option"
	default:
		return "UNK"
	}
}

type ActionType int

const (
	UndefinedAction ActionType = iota
	SellToOpen
	SellToClose
	BuyToOpen
	BuyToClose
	Sell
	Buy
)

func ActionTypeFromString(input string) ActionType {
	switch input {
	case "Sell to Open":
		return SellToOpen
	case "Sell to Close":
		return SellToClose
	case "Buy to Open":
		return BuyToOpen
	case "Buy to Close":
		return BuyToClose
	case "Sell":
		return Sell
	case "Buy":
		return Buy
	}

	return UndefinedAction
}

func (actionType ActionType) String() string {
	switch actionType {
	case SellToOpen:
		return "Sell to Open"
	case SellToClose:
		return "Sell to Close"
	case BuyToOpen:
		return "Buy to Open"
	case BuyToClose:
		return "Buy to Close"
	case Sell:
		return "Sell"
	case Buy:
		return "Buy"
	default:
		return "UNK"
	}
}

type ActionCondition int

const (
	UndefinedActionCondition ActionCondition = iota
	Route
	Cancel
)

func ActionConditionFromString(input string) ActionCondition {
	switch input {
	case "route":
		return Route
	case "cancel":
		return Cancel
	}

	return UndefinedActionCondition
}

func (actionCondition ActionCondition) String() string {
	switch actionCondition {
	case Route:
		return "route"
	case Cancel:
		return "cancel"
	default:
		return "UNK"
	}
}

type IndicatorType int

const (
	UndefinedIndicatorType IndicatorType = iota
	Last
	NAT
)

func IndicatorFromString(input string) IndicatorType {
	switch input {
	case "last":
		return Last
	case "nat":
		return NAT
	}

	return UndefinedIndicatorType
}

func (indicatorType IndicatorType) String() string {
	switch indicatorType {
	case Last:
		return "last"
	case NAT:
		return "nat"
	default:
		return "UNK"
	}
}

type ComparatorType int

const (
	UndefinedComparator ComparatorType = iota
	GTE
	LTE
)

func ComparatorFromString(input string) ComparatorType {
	switch input {
	case "gte":
		return GTE
	case "lte":
		return LTE
	}

	return UndefinedComparator
}

func (comparatorType ComparatorType) String() string {
	switch comparatorType {
	case GTE:
		return "gte"
	case LTE:
		return "lte"
	default:
		return "UNK"
	}
}

type Transaction struct {
	ID                               int64                `json:"id"`
	AccountNumber                    string               `json:"account-number"`
	ExecutedAt                       time.Time            `json:"executed-at"`
	TransactionDate                  time.Time            `json:"transaction-date"`
	TransactionType                  string               `json:"transaction-type"`
	TransactionSubType               string               `json:"transaction-sub-type"`
	Description                      string               `json:"description"`
	UnderlyingSymbol                 string               `json:"underlying-symbol"`
	InstrumentType                   InstrumentTypeChoice `json:"instrument-type"`
	Symbol                           string               `json:"symbol"`
	Action                           ActionType           `json:"action"`
	Quantity                         float64              `json:"quantity"`
	Price                            float64              `json:"price"`
	Value                            float64              `json:"value"`
	ValueEffect                      Effect               `json:"value-effect"`
	RegulatoryFees                   float64              `json:"regulatory-fees"`
	RegulatoryFeesEffect             Effect               `json:"regulatory-fees-effect"`
	ClearingFees                     float64              `json:"clearing-fees"`
	ClearingFeesEffect               Effect               `json:"clearing-fees-effect"`
	OtherCharge                      float64              `json:"other-charge"`
	OtherChargeEffect                Effect               `json:"other-charge-effect"`
	OtherChargeDescription           string               `json:"other-charge-description"`
	NetValue                         float64              `json:"net-value"`
	NetValueEffect                   Effect               `json:"net-value-effect"`
	Commission                       float64              `json:"commission"`
	CommissionEffect                 Effect               `json:"commission-effect"`
	ProprietaryIndexOptionFees       float64              `json:"proprietary-index-option-fees"`
	ProprietaryIndexOptionFeesEffect Effect               `json:"proprietary-index-option-fees-effect"`
	IsEstimatedFee                   bool                 `json:"is-estimated-fee"`
	OrderID                          int64                `json:"order-id"`
	Lots                             []*Lot               `json:"lots"`
	LegCount                         int64                `json:"leg-count"`
	DestinationVenue                 string               `json:"destination-venue"`
	AgencyPrice                      float64              `json:"agency-price"`
	PrincipalPrice                   float64              `json:"principal-price"`
	ExternalExchangeOrderNumber      string               `json:"ext-exchange-order-number"`
	ExternalGlobalOrderNumber        int64                `json:"ext-global-order-number"`
	ExternalGroupID                  string               `json:"ext-group-id"`
	ExternalGroupFillID              string               `json:"ext-group-fill-id"`
	ExternalExecutionID              string               `json:"ext-exec-id"`
	ExecutionID                      string               `json:"exec-id"`
	Exchange                         string               `json:"exchange"`
	ReversesID                       int64                `json:"reverses-id"`
	ExchangeAffiliationID            string               `json:"exchange-affiliation-identifier"`
	CostBasisReconciliationDate      time.Time            `json:"cost-basis-reconciliation-date"`
}

type Lot struct {
	ID                string    `json:"id"`
	TransactionID     int64     `json:"transaction-id"`
	Quantity          float64   `json:"quantity"`
	Price             float64   `json:"price"`
	QuantityDirection string    `json:"quantity-direction"`
	ExecutedAt        time.Time `json:"executed-at"`
	TransactionDate   time.Time `json:"transaction-date"`
}

type Order struct {
	// The length in time before the order expires. i.e. `Day`, `GTC`, `GTD`, `Ext`, `GTC Ext` or `IOC`
	TimeInForce string `json:"time-in-force"`

	// The date in which a GTD order will expire
	GTCDate time.Time `json:"gtc-date,omitempty"`

	// The type of order in regards to the price. i.e. `Limit`, `Market`, `Marketable Limit`, `Stop`, `Stop Limit`, `Notional Market`
	OrderType OrderTypeChoice `json:"order-type"`

	// The price trigger at which a stop or stop-limit order becomes valid
	StopTrigger float64 `json:"stop-trigger,omitempty"`

	// The price of the Order. Reuired for limit and stop-limit orders
	Price float64 `json:"price,omitempty"`

	// If pagy or receive payment for placing the order. i.e. `Credit` or `Debit`
	PriceEffect Effect `json:"price-effect,omitempty"`

	// The notional value of the Order, required for ntional market orders
	Value float64 `json:"value,omitempty"`

	// If pay or receive payment for placing the notional market order. i.e. Credit or Debit
	ValueEffect Effect `json:"value-effect,omitempty"`

	// The source the order is coming from
	Source string `json:"source,omitempty"`

	// Account partition key
	PartitionKey string `json:"parition-key,omitempty"`

	Legs []*Leg `json:"legs"`

	OrderRules *Rules `json:"rules,omitempty"`
}

type Leg struct {
	// The type of Instrument. i.e. `Cryptocurrency`, `Equity`, `Equity Offering`, `Equity Option`, `Future` or `Future Option`
	InstrumentType InstrumentTypeChoice `json:"instrument-type"`

	// The stock ticker symbol `AAPL, occ option symbol `AAPL 191004P00275000`, TW future symbol `/ESZ9`, or TW future option symbol `./ESZ9EW4U9 190927P2975`
	Symbol string `json:"symbol"`

	// The size of the contract. Required for all orders but notional market.
	Quantity int64 `json:"quantity"`

	// The directional action of the leg. i.e. Sell to Open, Sell to Close, Buy to Open, Buy to Close, Sell or Buy. Note: Buy and Sell are only applicable to Futures orders.
	Action ActionType `json:"action"`
}

type LegStatus struct {
	// The type of Instrument. i.e. `Cryptocurrency`, `Equity`, `Equity Offering`, `Equity Option`, `Future` or `Future Option`
	InstrumentType InstrumentTypeChoice `json:"instrument-type"`

	// The stock ticker symbol `AAPL, occ option symbol `AAPL 191004P00275000`, TW future symbol `/ESZ9`, or TW future option symbol `./ESZ9EW4U9 190927P2975`
	Symbol string `json:"symbol"`

	// The size of the contract. Required for all orders but notional market.
	Quantity string `json:"quantity"`

	RemainingQuantity string `json:"remaining-quantity"`

	// The directional action of the leg. i.e. Sell to Open, Sell to Close, Buy to Open, Buy to Close, Sell or Buy. Note: Buy and Sell are only applicable to Futures orders.
	Action ActionType `json:"action"`

	Fills []*FillStatus `json:"fills"`
}

type FillStatus struct {
	ExternalGroupFillID string    `json:"ext-group-fill-id"`
	ExternalExecutionID string    `json:"ext-exec-id"`
	FillID              string    `json:"fill-id"`
	Quantity            string    `json:"quantity"`
	FillPrice           float64   `json:"fill-price"`
	FilledAt            time.Time `json:"filled-at"`
	DestinationVenue    string    `json:"destination-venue"`
}

type Rules struct {
	// Earliest time an order should route at
	RouteAfter time.Time `json:"route-after,omitempty"`

	// Latest time an order should be canceled at
	CancelAt time.Time `json:"cancel-at,omitempty"`

	Conditions []*Condition `json:"conditions,omitempty"`
}

type RuleStatus struct {
	// Earliest time an order should route at
	RouteAfter time.Time `json:"route-after,omitempty"`

	RoutedAt time.Time `json:"routed-at"`

	// Latest time an order should be canceled at
	CancelAt time.Time `json:"cancel-at,omitempty"`

	CancelledAt time.Time `json:"cancelled-at"`

	Conditions []*ConditionStatus `json:"conditions,omitempty"`
}

type Condition struct {
	// The action in which the trigger is enacted, i.e. `route` and `cancel`
	Action ActionCondition `json:"action"`

	// The symbol to apply the condition to. I.e. Stock ticker symbol `AAPL` or the TW future symbol `/ESZ9`
	Symbol string `json:"symbol,omitempty"`

	// The instrument's type in relation to the condition. i.e. `Equity` or `Future`
	InstrumentType InstrumentTypeChoice `json:"instrument-type,omitempty"`

	// The indicator for the trigger
	Indicator IndicatorType `json:"indicator,omitempty"`

	// How to compare against the threshold. One of `gte` or `lte`
	Comparator ComparatorType `json:"comparator,omitempty"`

	// The price at which the condition triggers
	Threshold float64 `json:"threshold"`
}

type ConditionStatus struct {
	ID string `json:"id"`

	// The action in which the trigger is enacted, i.e. `route` and `cancel`
	Action ActionCondition `json:"action"`

	// Time the condition was triggered
	TriggeredAt time.Time `json:"triggered-at"`

	TriggeredValue float64 `json:"triggered-value"`

	// The symbol to apply the condition to. I.e. Stock ticker symbol `AAPL` or the TW future symbol `/ESZ9`
	Symbol string `json:"symbol,omitempty"`

	// The instrument's type in relation to the condition. i.e. `Equity` or `Future`
	InstrumentType InstrumentTypeChoice `json:"instrument-type,omitempty"`

	// The indicator for the trigger
	Indicator IndicatorType `json:"indicator,omitempty"`

	// How to compare against the threshold. One of `gte` or `lte`
	Comparator ComparatorType `json:"comparator,omitempty"`

	// The price at which the condition triggers
	Threshold float64 `json:"threshold"`

	IsThresholdBasedOnNotional bool `json:"is-threshold-based-on-notional"`

	PriceComponents []*ConditionPriceComponents `json:"price-components"`
}

type ConditionPriceComponents struct {
	Symbol            string               `json:"symbol"`
	InstrumentType    InstrumentTypeChoice `json:"instrument-type"`
	Quantity          string               `json:"quantity"`
	QuantityDirection string               `json:"quantity-direction"`
}

// OrderResponse contains the values returned from tastytrade after placing an order
type OrderResponse struct {
	Order               *OrderStatus       `json:"order"`
	EffectOnBuyingPower *BuyingPowerChange `json:"buying-power-effect"`
	FeeCalculation      *FeeInfo           `json:"fee-calculation"`
	Errors              []*ErrorMsg        `json:"errors"`
	Warnings            []*ErrorMsg        `json:"warnings"`
}

type BuyingPowerChange struct {
	ChangeInMarginRequirement            float64 `json:"change-in-margin-requirement"`
	ChangeInMarginRequirementEffect      Effect  `json:"change-in-margin-requirement-effect"`
	ChangeInBuyingPower                  float64 `json:"change-in-buying-power"`
	ChangeInBuyingPowerEffect            Effect  `json:"change-in-buying-power-effect"`
	CurrentBuyingPower                   float64 `json:"current-buying-power"`
	CurrentBuyingPowerEffect             Effect  `json:"current-buying-power-effect"`
	NewBuyingPower                       float64 `json:"new-buying-power"`
	NewBuyingPowerEffect                 Effect  `json:"new-buying-power-effect"`
	IsolatedOrderMarginRequirement       float64 `json:"isolated-order-margin-requirement"`
	IsolatedOrderMarginRequirementEffect Effect  `json:"isolated-order-margin-requirement-effect"`
	IsSpread                             bool    `json:"is-spread"`
	Impact                               float64 `json:"impact"`
	EffectOnCash                         Effect  `json:"effect"`
}

type FeeInfo struct {
	RegulatoryFees                   float64 `json:"regulatory-fees"`
	RegulatoryFeesEffect             Effect  `json:"regulatory-fees-effect"`
	ClearingFees                     float64 `json:"clearing-fees"`
	ClearingFeesEffect               Effect  `json:"clearing-fees-effect"`
	Commission                       float64 `json:"commission"`
	CommissionEffect                 Effect  `json:"commission-effect"`
	ProprietaryIndexOptionFees       float64 `json:"proprietary-index-option-fees"`
	ProprietaryIndexOptionFeesEffect Effect  `json:"proprietary-index-option-fees-effect"`
	TotalFees                        float64 `json:"total-fees"`
	TotalFeesEffect                  Effect  `json:"total-fees-effect"`
}

type OrderStatus struct {
	Size                     string               `json:"size"`
	TimeInForce              string               `json:"time-in-force"`
	TerminalAt               time.Time            `json:"terminal-at"`
	Editable                 bool                 `json:"editable"`
	ContingentStatus         string               `json:"contingent-status"`
	Legs                     []*LegStatus         `json:"legs"`
	GTCDate                  time.Time            `json:"gtc-date"`
	UpdatedAt                string               `json:"updated-at"`
	InFlightAt               time.Time            `json:"in-flight-at"`
	ReplacesOrderID          string               `json:"replaces-order-id"`
	UnderlyingSymbol         string               `json:"underlying-symbol"`
	Edited                   bool                 `json:"edited"`
	Price                    float64              `json:"price"`
	CancelUsername           string               `json:"cancel-username"`
	AccountNumber            string               `json:"account-number"`
	ConfirmationStatus       string               `json:"confirmation-status"`
	CancelUserID             string               `json:"cancel-user-id"`
	Cancellable              bool                 `json:"cancellable"`
	ValueEffect              Effect               `json:"value-effect"`
	StopTrigger              string               `json:"stop-trigger"`
	CancelledAt              time.Time            `json:"cancelled-at"`
	UnderlyingInstrumentType InstrumentTypeChoice `json:"underlying-instrument-type"`
	Value                    float64              `json:"value"`
	RejectReason             string               `json:"reject-reason"`
	Status                   string               `json:"status"`
	LiveAt                   time.Time            `json:"live-at"`
	PreflightID              string               `json:"preflight-id"`
	PriceEffect              Effect               `json:"price-effect"`
	Username                 string               `json:"username"`
	ReplacingOrderID         string               `json:"replacing-order-id"`
	ComplexOrderID           string               `json:"complex-order-id"`
	OrderType                OrderTypeChoice      `json:"order-type"`
	ID                       string               `json:"id"`
	OrderRule                []*RuleStatus        `json:"order-rule"`
	UserId                   string               `json:"user-id"`
	ComplexOrderTag          string               `json:"complex-order-tag"`
	ReceivedAt               time.Time            `json:"received-at"`
}

type ErrorMsg struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	PreflightID string `json:"preflight-id"`
}
