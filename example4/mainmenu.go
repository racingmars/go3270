// This file is part of https://github.com/racingmars/go3270/
// Copyright 2025 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

// This example demonstrates using the RunTransactions() approach to
// structuring a go3270 application.

// This file contains the main menu and the example placeholder features
// of the application.

package main

import (
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/racingmars/go3270"
)

const mainmenuOption = "option"
const mainmenuUsername = "username"
const mainmenuTime = "time"
const mainmenuUsers = "users"
const mainmenuError = "errormsg"

var mainmenuScreen = go3270.Screen{
	{Row: 0, Col: 31, Intense: true, Content: "Main Menu"},

	// Option
	{Row: 1, Col: 0, Content: "Option ===>", Color: go3270.Green},
	{Row: 1, Col: 12, Name: mainmenuOption, Write: true,
		Highlighting: go3270.Underscore, Color: go3270.Turquoise},
	{Row: 1, Col: 79, Autoskip: true}, // field "stop" character

	// Info
	{Row: 3, Col: 57, Content: "User ID . :", Color: go3270.Green},
	{Row: 3, Col: 69, Name: mainmenuUsername, Color: go3270.Turquoise},
	{Row: 4, Col: 57, Content: "Time. . . :", Color: go3270.Green},
	{Row: 4, Col: 69, Name: mainmenuTime, Color: go3270.Turquoise},
	{Row: 5, Col: 57, Content: "Users . . :", Color: go3270.Green},
	{Row: 5, Col: 69, Name: mainmenuUsers, Color: go3270.Turquoise},

	// Options
	{Row: 3, Col: 0, Content: "1", Color: go3270.White},
	{Row: 3, Col: 3, Content: "Feature 1",
		Color: go3270.Turquoise, Intense: true},
	{Row: 3, Col: 17, Content: "A very cool feature", Color: go3270.Green},

	{Row: 4, Col: 0, Content: "2", Color: go3270.White},
	{Row: 4, Col: 3, Content: "Feature 2",
		Color: go3270.Turquoise, Intense: true},
	{Row: 4, Col: 17, Content: "Another neat feature", Color: go3270.Green},

	{Row: 5, Col: 0, Content: "3", Color: go3270.White},
	{Row: 5, Col: 3, Content: "Feature 3",
		Color: go3270.Turquoise, Intense: true},
	{Row: 5, Col: 17, Content: "This one's a boring feature",
		Color: go3270.Green},

	{Row: 7, Col: 5, Content: "Enter", Color: go3270.Green},
	{Row: 7, Col: 11, Content: "X", Color: go3270.Turquoise, Intense: true},
	{Row: 7, Col: 13, Content: "to log off and exit.", Color: go3270.Green},

	// Error message
	{Row: 21, Col: 0, Name: mainmenuError, Color: go3270.Red, Intense: true},

	// Key legend
	{Row: 23, Col: 1, Content: "F1=Help"},
	{Row: 23, Col: 14, Content: "F3=Exit"},
	{Row: 23, Col: 27, Content: "F5=Refresh"},
}

type mainmenuData struct {
	option string // pre-populated option field value
	errmsg string // error message to display
}

// mainmenu transaction accepts a mainmenuData struct as the data if the
// option field or error message should be populated.
func (sess *session) mainmenu(conn net.Conn, devinfo go3270.DevInfo, data any) (
	go3270.Tx, any, error) {

	fieldValues := make(map[string]string)

	if data != nil {
		if d, ok := data.(mainmenuData); ok {
			fieldValues[mainmenuOption] = d.option
			fieldValues[mainmenuError] = d.errmsg
		}
	}

	fieldValues[mainmenuUsername] = strings.ToUpper(sess.user.Username)
	fieldValues[mainmenuTime] = time.Now().UTC().Format("15:04:05")
	fieldValues[mainmenuUsers] = strconv.Itoa(sess.g.usercount)

	resp, err := go3270.HandleScreen(
		mainmenuScreen,                // the screen to display
		nil,                           // (no) rules to enforce
		fieldValues,                   // pre-populated values in fields
		[]go3270.AID{go3270.AIDEnter}, // keys we accept -- validating
		[]go3270.AID{ // keys we accept -- non-validating
			go3270.AIDPF1,
			go3270.AIDPF3,
			go3270.AIDPF5,
		},
		mainmenuError, // name of field to put error messages in
		1, 13,         // cursor coordinates
		conn,
		devinfo.Codepage())
	if err != nil {
		return nil, nil, err
	}

	switch resp.AID {
	case go3270.AIDPF1:
		// Display help, then return to this transaction
		return help(sess.mainmenu), nil, nil
	case go3270.AIDPF3:
		// Exit
		return nil, nil, nil
	case go3270.AIDPF5:
		// Refresh
		return sess.mainmenu, nil, nil
	}

	// Handle the available options:
	switch resp.Values[mainmenuOption] {
	case "1":
		return sess.exampleFeature,
			"This is feature 1, a very cool feature.", nil
	case "2":
		return sess.exampleFeature,
			"This is feature 2, another neat feature.", nil
	case "3":
		return sess.exampleFeature,
			"This is feature 3, which is a boring one.", nil
	case "x", "X":
		// As if user hit PF3; exit
		return nil, nil, nil
	case "":
		// As if user hit PF5, refresh
		return sess.mainmenu, nil, nil
	default:
		// Run the transaction again with an error message
		return sess.mainmenu,
			mainmenuData{
				option: resp.Values[mainmenuOption],
				errmsg: "Unknown option",
			}, nil
	}
}

const featureMessage = "message"

var exampleScreen = go3270.Screen{
	{Row: 0, Col: 30, Intense: true, Content: "Application Feature"},

	// Info -- replicate the data shown on the main menu
	{Row: 3, Col: 57, Content: "User ID . :", Color: go3270.Green},
	{Row: 3, Col: 69, Name: mainmenuUsername, Color: go3270.Turquoise},
	{Row: 4, Col: 57, Content: "Time. . . :", Color: go3270.Green},
	{Row: 4, Col: 69, Name: mainmenuTime, Color: go3270.Turquoise},
	{Row: 5, Col: 57, Content: "Users . . :", Color: go3270.Green},
	{Row: 5, Col: 69, Name: mainmenuUsers, Color: go3270.Turquoise},

	// Feature-specific message
	{Row: 12, Col: 0, Name: featureMessage},

	{Row: 14, Col: 0, Content: "Press"},
	{Row: 14, Col: 6, Content: "PF3", Intense: true, Color: go3270.White},
	{Row: 14, Col: 10, Content: "to return to the main menu."},

	// Error message
	{Row: 21, Col: 0, Name: mainmenuError, Color: go3270.Red, Intense: true},

	// Key legend
	{Row: 23, Col: 1, Content: "F1=Help"},
	{Row: 23, Col: 14, Content: "F3=Exit"},
	{Row: 23, Col: 27, Content: "F5=Refresh"},
}

// exampleFeature is a transaction that will act as a placeholder for real
// application functionality. It accepts a string in the data which will
// be displayed on the panel.
func (sess *session) exampleFeature(conn net.Conn, devinfo go3270.DevInfo,
	data any) (go3270.Tx, any, error) {

	fieldValues := make(map[string]string)

	if data != nil {
		if message, ok := data.(string); ok {
			fieldValues[featureMessage] = message
		}
	}

	fieldValues[mainmenuUsername] = strings.ToUpper(sess.user.Username)
	fieldValues[mainmenuTime] = time.Now().UTC().Format("15:04:05")
	fieldValues[mainmenuUsers] = strconv.Itoa(sess.g.usercount)

	resp, err := go3270.HandleScreen(
		exampleScreen, // the screen to display
		nil,           // (no) rules to enforce
		fieldValues,   // pre-populated values in fields
		nil,           // keys we accept -- validating
		[]go3270.AID{ // keys we accept -- non-validating
			go3270.AIDPF1,
			go3270.AIDPF3,
			go3270.AIDPF5,
		},
		mainmenuError, // name of field to put error messages in
		23, 79,        // cursor coordinates
		conn,
		devinfo.Codepage())
	if err != nil {
		return nil, nil, err
	}

	switch resp.AID {
	case go3270.AIDPF1:
		// Display help, then return to this transaction
		return help(sess.exampleFeature), data, nil
	case go3270.AIDPF3:
		// Exit
		return sess.mainmenu, nil, nil
	case go3270.AIDPF5:
		// Refresh
		return sess.exampleFeature, data, nil
	}

	// ...there shouldn't be any actions not handled above. Just in case,
	// we'll just re-run this transaction as if refresh was hit.
	return sess.exampleFeature, data, nil
}
