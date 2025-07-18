// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/racingmars/go3270"
)

func init() {
	// put the go3270 library in debug mode
	go3270.Debug = os.Stderr
}

// A Screen is an array of go3270.Field structs. We will build two screens,
// a query screen and a result screen:

var screen1 = go3270.Screen{
	{Row: 0, Col: 27, Intense: true, Content: "3270 Example Application"},
	{Row: 2, Col: 0, Content: "Welcome to the go3270 example application. Please enter your name."},
	{Row: 4, Col: 0, Content: "First Name  . . ."},
	{Row: 4, Col: 19, Name: "fname", Write: true, Highlighting: go3270.Underscore},
	{Row: 4, Col: 40, Autoskip: true}, // field "stop" character
	{Row: 5, Col: 0, Content: "Last Name . . . ."},
	{Row: 5, Col: 19, Name: "lname", Write: true, Highlighting: go3270.Underscore},
	{Row: 5, Col: 40, Autoskip: true}, // field "stop" character,
	{Row: 6, Col: 0, Content: "Password  . . . ."},
	{Row: 6, Col: 19, Name: "password", Write: true, Hidden: true},
	{Row: 6, Col: 40, Autoskip: true}, // field "stop" character,
	{Row: 7, Col: 0, Content: "Change me  . . ."},
	{Row: 7, Col: 19, Name: "changeme", Content: "change me", Write: true, Highlighting: go3270.Underscore},
	{Row: 7, Col: 40}, // field "stop" character
	{Row: 8, Col: 0, Content: "Press"},
	{Row: 8, Col: 6, Intense: true, Content: "enter"},
	{Row: 8, Col: 12, Content: "to submit your name."},
	{Row: 10, Col: 0, Intense: true, Color: go3270.Red, Name: "errormsg"}, // a blank field for error messages
	{Row: 22, Col: 0, Content: "PF3 Exit"},
}

var screen1rules = go3270.Rules{
	"fname":    {Validator: go3270.NonBlank},
	"lname":    {Validator: go3270.NonBlank},
	"password": {Validator: go3270.NonBlank, Reset: true},
	"changeme": {Validator: go3270.NonBlank, MustChange: true, ErrorText: "You can't leave \"change me\" as the value in the Change me field"},
}

var screen2 = go3270.Screen{
	{Row: 0, Col: 27, Intense: true, Content: "3270 Example Application"},
	{Row: 2, Col: 0, Content: "Thank you for submitting your name. Here's what I know:"},
	{Row: 4, Col: 0, Content: "Your first name is"},
	{Row: 4, Col: 19, Name: "fname"}, // We're giving this field a name to replace its value at runtime
	{Row: 5, Col: 0, Content: "And your last name is"},
	{Row: 5, Col: 22, Name: "lname"}, // We're giving this field a name to replace its value at runtime
	{Row: 6, Col: 0, Name: "passwordOutput"},
	{Row: 8, Col: 0, Content: "Press"},
	{Row: 8, Col: 6, Intense: true, Content: "enter"},
	{Row: 8, Col: 12, Content: "to enter your name again, or"},
	{Row: 8, Col: 41, Intense: true, Content: "PF3"},
	{Row: 8, Col: 45, Content: "to quit and disconnect."},
	{Row: 11, Col: 0, Color: go3270.Turquoise, Highlighting: go3270.ReverseVideo, Content: "Here is a field with extended attributes."},
	{Row: 11, Col: 42}, // remember to "stop" fields with a regular field, to clear the reverse video for example
	{Row: 20, Col: 0, Intense: true, Color: go3270.Red, Name: "errormsg"}, // a blank field for error messages
	{Row: 22, Col: 0, Content: "PF3 Exit"},
}

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
	if _, err := go3270.NegotiateTelnet(conn); err != nil {
		fmt.Println(err)
		return
	}

	fieldValues := make(map[string]string)

	// We will loop forever until the user quits
	for {
		// First we display screen 1. The HandleScreen() function handles
		// input validation and processing attention keys we don't handle.
		response, err := go3270.HandleScreen(
			screen1,                       // the screen to display
			screen1rules,                  // the rules to enforce
			fieldValues,                   // any field values we wish to supply
			[]go3270.AID{go3270.AIDEnter}, // the AID keys we support
			[]go3270.AID{go3270.AIDPF3},   // keys that are "exit" keys
			"errormsg",                    // the field to write error message into
			4, 20,                         // the row and column to place the cursor
			conn)
		if err != nil {
			fmt.Println(err)
			return
		}

		if response.AID == go3270.AIDPF3 {
			// Exit
			break
		}

		fieldValues = response.Values

		// Now we're ready to display screen2
		passwordLength := len(strings.TrimSpace(fieldValues["password"]))
		passwordPlural := "s"
		if passwordLength == 1 {
			passwordPlural = ""
		}
		fieldValues["passwordOutput"] = fmt.Sprintf("Your password was %d character%s long",
			passwordLength, passwordPlural)

		response, err = go3270.HandleScreen(
			screen2,                       // the screen to display
			nil,                           // the rules to enforce
			fieldValues,                   // any field values we wish to supply
			[]go3270.AID{go3270.AIDEnter}, // the AID keys we support
			[]go3270.AID{go3270.AIDPF3},   // keys that are "exit" keys
			"errormsg",                    // the field to write error message into
			0, 0,                          // the row and column to place the cursor
			conn)
		if err != nil {
			fmt.Println(err)
			return
		}

		// If the user pressed PF3, exit
		if response.AID == go3270.AIDPF3 {
			break
		}

		// If they pressed anything else, just let the loop continue...
		continue
	}

	fmt.Println("Connection closed")
}
