// This file is part of https://github.com/racingmars/go3270/
// Copyright 2025 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

// This example demonstrates using the RunTransactions() approach to
// structuring a go3270 application.

// This file contains the help transaction generator.

package main

import (
	"net"

	"github.com/racingmars/go3270"
)

var helpScreen = go3270.Screen{
	{Row: 0, Col: 45, Intense: true, Content: "Online Help"},

	{Row: 2, Col: 0,
		Content: "This help screen is an example of a transaction that"},
	{Row: 3, Col: 0,
		Content: "can be used as the next transaction from many other transactions"},
	{Row: 4, Col: 0, Content: "and returns to the transaction that called it."},

	{Row: 6, Col: 0, Content: "Press"},
	{Row: 6, Col: 6, Content: "PF3", Color: go3270.White, Intense: true},
	{Row: 6, Col: 10,
		Content: "to return to the transaction from which you came."},

	// Error message
	{Row: 21, Col: 0, Name: "errormsg", Color: go3270.Red, Intense: true},

	// Key legend
	{Row: 23, Col: 1, Content: "F3=Exit"},
}

// The help transaction is an example of returning a closure over a parameter,
// in this case the desired next transaction. This allows us to use the same
// transaction from multiple other transactions and return to the requested
// transaction when the user leaves the help transaction.
//
// Any data passed to help will be passed back to the transaction it returns
// to.
//
// The help transaction has no need for any global or session state, so this
// generator function for it is a stand-alone function and not a method on the
// session.
func help(returnTransaction go3270.Tx) go3270.Tx {
	return func(conn net.Conn, data any) (go3270.Tx, any, error) {
		_, err := go3270.HandleScreen(
			helpScreen,                  // the screen to display
			nil,                         // (no) rules to enforce
			nil,                         // pre-populated values in fields
			[]go3270.AID{go3270.AIDPF3}, // keys we accept
			nil,
			"errormsg", // name of field to put error messages in
			23, 79,     // cursor coordinates
			conn)
		if err != nil {
			return nil, nil, err
		}

		// Any accepted key returns to the requested returnTransaction
		return returnTransaction, data, nil
	}
}
