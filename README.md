# go-tasty


[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/penny-vault/go-tasty) [![license](http://img.shields.io/badge/license-Apache-red.svg?style=flat)](https://opensource.org/license/apache-2-0/) [![Build Status](https://github.com/penny-vault/go-tasty/actions/workflows/test.yml/badge.svg)](https://github.com/penny-vault/go-tasty/actions/workflows/test.yml)

go-tasty is an idiomatic go library for interfacing with the [tastytrade Open API](https://support.tastytrade.com/support/s/solutions/articles/43000700385). To use the library you will need an account with tastytrade that is opted-in to the API. See the instructions under `Open API Access` on the [tastytrade Open API help page](https://support.tastytrade.com/support/s/solutions/articles/43000700385). For more information about the api see https://developer.tastytrade.com/

## Features

* Download account information
* Place and monitor trades

## Todo

Currently go-tasty doesn't support every portion of the tastytrade Open API. The following
endpoints need to be implemented:

* Instruments
* Margin Requirements
* Market Metrics
* Net Liquidating Value History
* Risk Parameters
* Symbol Search
* Watchlists

Streaming account data and market data are not supported.

Finally, order management is limited to creating, listing, and deleting simple orders.
Complex order types for BLAST, OCO, OTO, OTOCO, and PAIRS are not supported.

## Installation

```bash
go get -u github.com/penny-vault/go-tasty
```

## Getting Started

### Simple example

```go
package main

import (
    "fmt"
    "github.com/penny-vault/go-tasty"
)

func main() {
    // Create a new session with the provided username and password
    // NewSession(login, password string, opts ...SessionOptions)
    session, err := gotasty.NewSession("<username-or-email>", "<password>", SessionOpts{
        RememberMe: true,
        Sandbox: true,
    })
    if err != nil {
        panic(err.Error())
    }

    // destroy the session
    if err := session.Delete(); err != nil {
        panic(err.Error())
    }
}
```
