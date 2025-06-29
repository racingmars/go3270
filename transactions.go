// This file is part of https://github.com/racingmars/go3270/
// Copyright 2025 by Matthew R. Wilson, licensed under the MIT license.
// See LICENSE in the project root for license information.

package go3270

import "net"

// Tx is a function that serves as one transaction in a go3270 application.
// The Tx function is called with the network connection to the client, the
// DevInfo for use with alternate screen writes, and a "data" value provided
// by the previous transaction. Tx functions return the next transaction to
// run (or nil to indicate the RunTransactions() function should terminate),
// the data to pass into the next transaction, and any error. If the error is
// non-nil, the RunTransactions() function will terminate and return the err.
// A non-nil error is _not_ passed between transactions, it terminates
// transaction processing.
type Tx func(conn net.Conn, dev DevInfo, data any) (
	next Tx, newdata any, err error)

// RunTransactions begins running transaction functions, starting with the
// initial transaction, until a transaction eventually returns nil for the
// next transaction, or until a transaction function returns a non-nil error
// value. data (which may be nil, if the initial transaction does not require
// data) is passed in as the data to the initial transaction.
//
// dev is the DevInfo of the connected client, as obtained from
// NegotiateTelnet(). It is safe to pass a nil DevInfo, in which case all
// transactions will only be able to operate with the default 24x80 screen
// size.
func RunTransactions(conn net.Conn, dev DevInfo, initial Tx,
	data any) error {

	var next Tx
	var err error

	next = initial

	if dev == nil {
		dev = &deviceInfo{rows: 24, cols: 80, termtype: "DEFAULT"}
	}

	// We run transactions until there isn't a next transaction to run, or
	// an error.
	for {
		next, data, err = next(conn, dev, data)
		if err != nil {
			// Error means we bail out and return the error to the caller.
			return err
		}

		if next == nil {
			// nil next transaction means we're done.
			return nil
		}
	}
}
