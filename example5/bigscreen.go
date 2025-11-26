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

var biglayout = go3270.Screen{
	// Column will be calculated at runtime for the following field:
	{Row: 0, Intense: true, Content: "3270 Screen Size Example"},

	{Row: 2, Col: 0,
		Content: "This screen is using the full size that your terminal supports."},

	{Row: 4, Col: 0, Content: "Terminal Type  . . ."},
	{Row: 4, Col: 21, Name: "termtype", Intense: true},

	{Row: 4, Col: 40, Content: "Code page . . ."},
	{Row: 4, Col: 56, Name: "codepage", Intense: true},

	{Row: 5, Col: 0, Content: "Rows . . . . . . . ."},
	{Row: 5, Col: 21, Name: "rows", Intense: true},

	{Row: 6, Col: 0, Content: "Columns  . . . . . ."},
	{Row: 6, Col: 21, Name: "cols", Intense: true},

	{Row: 8, Col: 0, Content: "To visit a default sized screen, press"},
	{Row: 8, Col: 39, Content: "PF1", Color: go3270.Yellow, Intense: true},

	{Row: 9, Col: 0, Content: "To exit and disconnect, press"},
	{Row: 9, Col: 30, Content: "PF3", Color: go3270.Yellow, Intense: true},

	// a blank field for error messages
	{Row: 11, Col: 0, Intense: true, Color: go3270.Red, Name: "errormsg"},
}

func bigscreen(conn net.Conn, devinfo go3270.DevInfo, data any) (
	go3270.Tx, any, error) {

	rows, cols := devinfo.AltDimensions()
	termtype := devinfo.TerminalType()
	codepage := "(unknown)"
	if devinfo.Codepage() != nil {
		codepage = devinfo.Codepage().ID()
	}

	// Make a local copy of the screen definition that we can append lines to.
	screen := make(go3270.Screen, len(biglayout))
	copy(screen, biglayout)

	// Center the title on any screen width
	screen[0].Col = (cols / 2) - (len(biglayout[0].Content) / 2)

	// We'll start writing "data lines" at row 13 up to the penultimate row on
	// the terminal
	for i := 13; i < rows-1; i++ {
		newfield := go3270.Field{Row: i, Col: 0,
			Content: fmt.Sprintf("This is data row %d.", i-12)}
		if i == rows-2 {
			newfield.Content += " (The last.)"
		}
		screen = append(screen, newfield)

		// And demonstrate that we can position to the full width, too.
		newfield = go3270.Field{Row: i, Col: cols - 5, Content: "<**>"}
		screen = append(screen, newfield)
	}

	// And an input field on the last row, to make sure field buffer address
	// decoding works on larger screens.
	newfield := go3270.Field{Row: rows - 1, Col: 0,
		Content: "Enter data here:", Color: go3270.Pink, Intense: true}
	screen = append(screen, newfield)
	newfield = go3270.Field{Row: rows - 1, Col: 17, Name: "inputdata",
		Write: true}
	screen = append(screen, newfield)
	newfield = go3270.Field{Row: rows - 1, Col: cols - 1} // "stop" field
	screen = append(screen, newfield)

	fieldValues := map[string]string{
		"termtype": termtype,
		"codepage": codepage,
		"rows":     strconv.Itoa(rows),
		"cols":     strconv.Itoa(cols),
	}

	if data != nil {
		fieldValues["errormsg"] = fmt.Sprintf("You said: %s", data.(string))
	}

	resp, err := go3270.HandleScreenAlt(
		screen,      // the screen to display
		nil,         // (no) rules to enforce
		fieldValues, // pre-populated values in fields
		[]go3270.AID{ // keys we accept -- validating
			go3270.AIDEnter,
		},
		[]go3270.AID{ // keys we accept -- non-validating
			go3270.AIDPF1,
			go3270.AIDPF3,
		},
		"errormsg", // name of field to put error messages in
		rows-1, 18, // cursor coordinates
		conn,               // network connection
		devinfo,            // device info for alternate screen size support
		devinfo.Codepage(), // client code page
	)
	if err != nil {
		return nil, nil, err
	}

	switch resp.AID {
	case go3270.AIDEnter:
		// Re-run current transaction, echoing back input
		return bigscreen, resp.Values["inputdata"], err
	case go3270.AIDPF1:
		// Go to default screen size transaction
		return normalscreen, nil, nil
	case go3270.AIDPF3:
		// Exit
		return nil, nil, nil
	default:
		// re-run current transaction
		return bigscreen, nil, nil
	}
}
