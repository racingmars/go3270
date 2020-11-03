// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

package main

import (
	"fmt"
	"net"
	"os"

	"github.com/racingmars/go3270"
)

func init() {
	go3270.Debug = os.Stderr
}

// A Screen is an array of go3270.Field structs:
var loginScreen = go3270.Screen{
	{Row: 0, Col: 30, Intense: true, Content: "3270 Example Screen"},
	{Row: 1, Col: 0, Content: "First Name  . . ."},
	{Row: 1, Col: 18, Name: "fname", Write: true, Content: "Test"},
	{Row: 2, Col: 0, Content: "Last Name . . . ."},
	{Row: 2, Col: 18, Name: "lname", Write: true},
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
	_, err := go3270.ShowScreen(loginScreen, nil, 1, 19, conn)
	if err != nil {
		panic(err)
	}

	fmt.Println("Connection closed")

}
