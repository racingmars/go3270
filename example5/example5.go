// This file is part of https://github.com/racingmars/go3270/
// Copyright 2025 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

// Example 5 demonstrates support for larger-than-default alternate screen
// sizes in terminals that are larger than 24x80.

package main

import (
	"fmt"
	"net"

	"github.com/racingmars/go3270"
)

func main() {
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
		go handle(conn)
	}
}

// handle is the handler for individual user connections.
func handle(conn net.Conn) {
	defer conn.Close()

	// Always begin new connection by negotiating the telnet options
	devinfo, err := go3270.NegotiateTelnet(conn)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = go3270.RunTransactions(conn, devinfo, bigscreen, nil)
	if err != nil {
		fmt.Println(err)
	}
}
