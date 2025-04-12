// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

// This example demonstrates updating portions of the screen while waiting
// for a client response.

package main

import (
	"fmt"
	"net"
	"time"

	"github.com/racingmars/go3270"
)

var screen = go3270.Screen{
	{Row: 0, Col: 35, Intense: true, Content: "3270 Clock"},
	{Row: 1, Col: 0, Color: go3270.White,
		Content: "------------------------------------------------------------------------------"},
	{Row: 5, Col: 5, Color: go3270.Turquoise, Content: "The current UTC time is:"},
	{Row: 5, Col: 30, Color: go3270.Yellow, Intense: true, Content: "XX:XX:XX"},
	{Row: 22, Col: 0, Content: "PF3 Exit"},
}

var refresh = go3270.Screen{
	screen[3], // Copy the time field that we want to update
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
	go3270.NegotiateTelnet(conn)

	// First, let's send the initial screen and wait forever for the user to
	// press PF3, and when we get it, send a message on the done channel.
	done := func() chan bool {
		done := make(chan bool)

		// This will run in a goroutine so it can block waiting for the
		// response, while the rest of the code below (which refreshes the
		// screen every second) can continue to run.
		go func() {
			// Loop forever, sending the background screen until user exits.
			for {
				screen[3].Content = time.Now().UTC().Format("15:04:05")
				response, err := go3270.ShowScreenOpts(screen, nil, conn,
					go3270.ScreenOpts{CursorRow: 23, CursorCol: 0})
				if err != nil {
					// User dropped connection, maybe? We'll end things.
					done <- true
					return
				}

				if response.AID == go3270.AIDPF3 {
					// User wants to quit.
					done <- true
					return
				}
			}
		}()

		return done
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			// User pressed PF3 (or we need to quit for some other reason)
			return

		case <-ticker.C:
			// Send the updated time, without clearing the screen
			refresh[0].Content = time.Now().UTC().Format("15:04:05")
			_, err := go3270.ShowScreenOpts(refresh, nil, conn,
				go3270.ScreenOpts{NoClear: true, NoResponse: true})
			if err != nil {
				// Bail out
				return
			}
		}
	}
}
