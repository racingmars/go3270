// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

package main

import (
	"fmt"
	"net"

	"github.com/racingmars/go3270"
)

var loginScreen = go3270.Screen{
	{Row: 0, Col: 0, Intense: true, Content: "Testing . . ."},
	{Row: 1, Col: 0, Content: "Name    . . ."},
	{Row: 1, Col: 14, Name: "name", Write: true},
}

func main() {
	ln, err := net.Listen("tcp", ":3270")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	go3270.NegotiateTelnet(conn)
	err := go3270.WriteScreen(loginScreen, 10, 15, conn)
	if err != nil {
		panic(err)
	}

	for {
		rbuf := make([]byte, 255)
		n, err := conn.Read(rbuf)
		if err != nil {
			break
		}
		for i := 0; i < n; i++ {
			fmt.Printf("%x", rbuf[i])
		}
		fmt.Printf("\n")
	}

}
