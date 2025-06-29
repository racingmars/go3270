// This file is part of https://github.com/racingmars/go3270/
// Copyright 2025 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

package main

import (
	"fmt"
	"net"
	"strconv"

	"github.com/racingmars/go3270"
)

var normallayout = go3270.Screen{
	{Row: 0, Col: 28, Intense: true, Content: "3270 Screen Size Example"},

	{Row: 2, Col: 0,
		Content: "This screen is using the default size that all terminals support, 24x80."},
	{Row: 3, Col: 0,
		Content: "But I know the following information about your particular terminal:"},

	{Row: 4, Col: 0, Content: "Terminal Type  . . ."},
	{Row: 4, Col: 21, Name: "termtype", Intense: true},

	{Row: 5, Col: 0, Content: "Rows . . . . . . . ."},
	{Row: 5, Col: 21, Name: "rows", Intense: true},
	{Row: 5, Col: 28, Content: "(but currently using 24)"},

	{Row: 6, Col: 0, Content: "Columns  . . . . . ."},
	{Row: 6, Col: 21, Name: "cols", Intense: true},
	{Row: 6, Col: 28, Content: "(but currently using 80)"},

	{Row: 8, Col: 0, Content: "To visit a large sized screen, press"},
	{Row: 8, Col: 37, Content: "PF1", Color: go3270.Yellow, Intense: true},

	{Row: 9, Col: 0, Content: "To exit and disconnect, press"},
	{Row: 9, Col: 30, Content: "PF3", Color: go3270.Yellow, Intense: true},

	// a blank field for error messages
	{Row: 11, Col: 0, Intense: true, Color: go3270.Red, Name: "errormsg"},
}

func normalscreen(conn net.Conn, devinfo go3270.DevInfo, data any) (
	go3270.Tx, any, error) {

	rows, cols := devinfo.AltDimensions()
	termtype := devinfo.TerminalType()

	// Make a local copy of the screen definition that we can append lines to.
	screen := make(go3270.Screen, len(normallayout))
	copy(screen, normallayout)

	// We'll start writing "data lines" at row 13 up to 24
	for i := 13; i < 24; i++ {
		newfield := go3270.Field{Row: i, Col: 0,
			Content: fmt.Sprintf("This is data row %d.", i-12)}
		if i == 23 {
			newfield.Content += " (The last.)"
		}
		screen = append(screen, newfield)

		newfield = go3270.Field{Row: i, Col: 80 - 5, Content: "<**>"}
		screen = append(screen, newfield)
	}

	fieldValues := map[string]string{
		"termtype": termtype,
		"rows":     strconv.Itoa(rows),
		"cols":     strconv.Itoa(cols),
	}

	// We can call the old HandleScreen(), or we could have used the new
	// HandleScreenAlt() and provided a nil DevInfo.
	resp, err := go3270.HandleScreen(
		screen,      // the screen to display
		nil,         // (no) rules to enforce
		fieldValues, // pre-populated values in fields
		nil,         // keys we accept -- validating
		[]go3270.AID{ // keys we accept -- non-validating
			go3270.AIDPF1,
			go3270.AIDPF3,
		},
		"errormsg", // name of field to put error messages in
		1, 1,       // cursor coordinates
		conn, // network connection
	)
	if err != nil {
		return nil, nil, err
	}

	switch resp.AID {
	case go3270.AIDPF1:
		// Go to big screen size transaction
		return bigscreen, nil, nil
	case go3270.AIDPF3:
		// Exit
		return nil, nil, nil
	default:
		// re-run current transaction
		return normalscreen, nil, nil
	}
}
