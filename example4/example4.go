// This file is part of https://github.com/racingmars/go3270/
// Copyright 2025 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

// This example demonstrates using the RunTransactions() approach to
// structuring a go3270 application.

package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/racingmars/go3270"
)

type global struct {
	usercount int // number of connected users
	countlock sync.Mutex
}

// session is the structure which holds global state for a user session of our
// application. The various Transactions will be methods on this struct.
type session struct {
	db   DB
	user User

	// We can also share global state between users
	g *global
}

func main() {
	// "Connect" to our database (for this example, it's just a mock in-memory
	// database).
	dbconn := Connect()

	gblstate := new(global)

	ln, err := net.Listen("tcp", ":3270")
	if err != nil {
		panic(err)
	}
	fmt.Println("LISTENING ON PORT 3270 FOR CONNECTIONS")
	fmt.Println("Press Ctrl-C to end server.")
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go handle(conn, dbconn, gblstate)
	}
}

// handle is the handler for individual user connections.
func handle(conn net.Conn, db DB, gblstate *global) {
	defer conn.Close()

	// Create the state for this client's connection
	state := session{
		db: db,
		g:  gblstate,
	}

	// Add to the global user counter
	gblstate.countlock.Lock()
	gblstate.usercount++
	gblstate.countlock.Unlock()

	// When the session ends, reduce the user count
	defer func() {
		gblstate.countlock.Lock()
		gblstate.usercount--
		gblstate.countlock.Unlock()
	}()

	// Always begin new connection by negotiating the telnet options
	go3270.NegotiateTelnet(conn)
	err := go3270.RunTransactions(conn, state.login, nil)
	if err != nil {
		fmt.Println(err)
	}
}
