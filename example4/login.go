// This file is part of https://github.com/racingmars/go3270/
// Copyright 2025 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

// This example demonstrates using the RunTransactions() approach to
// structuring a go3270 application.

// This file contains the login and user registration transactions.

package main

import (
	"net"

	"github.com/racingmars/go3270"
)

const loginUsername = "username"
const loginPassword = "password"
const loginPasswordConf = "password confirmation"
const loginName = "name"
const loginErr = "errormsg"

var loginScreen = go3270.Screen{
	{Row: 0, Col: 37, Intense: true, Content: "Logon"},
	{Row: 2, Col: 0,
		Content: "Welcome to the go3270 example application. Please log on."},

	// Username
	{Row: 4, Col: 0, Content: "Username . . .", Color: go3270.Green},
	{Row: 4, Col: 15, Name: loginUsername, Write: true,
		Highlighting: go3270.Underscore, Color: go3270.Turquoise},
	{Row: 4, Col: 24, Autoskip: true}, // field "stop" character

	// Password
	{Row: 5, Col: 0, Content: "Password . . .", Color: go3270.Green},
	{Row: 5, Col: 15, Name: loginPassword, Write: true, Hidden: true},
	{Row: 5, Col: 79}, // field "stop" character

	// Registration instructions
	{Row: 7, Col: 0, Content: "If you don't yet have an account, press"},
	{Row: 7, Col: 40, Content: "PF5", Color: go3270.White, Intense: true},
	{Row: 7, Col: 44, Content: "to register a new account."},

	// Error message
	{Row: 21, Col: 0, Name: loginErr, Color: go3270.Red, Intense: true},

	// Key legend
	{Row: 23, Col: 1, Content: "F1=Help"},
	{Row: 23, Col: 14, Content: "F3=Exit"},
	{Row: 23, Col: 27, Content: "F5=Register"},
}

var loginScreenRules = go3270.Rules{
	loginUsername: {Validator: go3270.NonBlank},
	loginPassword: {Validator: go3270.NonBlank, Reset: true},
}

// login transaction accepts a string value in data if the login screen
// should be initialized with an error message.
func (sess *session) login(conn net.Conn, _ go3270.DevInfo,
	data any) (go3270.Tx, any, error) {

	fieldValues := make(map[string]string)

	if data != nil {
		if errmsg, ok := data.(string); ok {
			fieldValues[loginErr] = errmsg
		}
	}

	resp, err := go3270.HandleScreen(
		loginScreen,                   // the screen to display
		loginScreenRules,              // the rules to enforce
		fieldValues,                   // pre-populated values in fields
		[]go3270.AID{go3270.AIDEnter}, // keys we accept -- validating
		[]go3270.AID{ // keys we accept -- non-validating
			go3270.AIDPF1,
			go3270.AIDPF3,
			go3270.AIDPF5,
			go3270.AIDClear},
		loginErr, // name of field to put error messages in
		4, 16,    // cursor coordinates
		conn)
	if err != nil {
		return nil, nil, err
	}

	switch resp.AID {
	case go3270.AIDClear:
		// re-run the transaction with empty values
		return sess.login, nil, nil
	case go3270.AIDPF1:
		// Display help transaction, then return to this transaction
		return help(sess.login), nil, nil
	case go3270.AIDPF3:
		// User wants to quit; return no next transaction
		return nil, nil, nil
	case go3270.AIDPF5:
		// Send user to registration transaction
		return sess.newuser, nil, nil
	}

	// If we didn't get one of the other allowed keys, try to log the user in.
	username := resp.Values[loginUsername]
	password := resp.Values[loginPassword]

	user, err := sess.db.GetUser(username)
	// NOTE: this is just an example application. Obviously in a real
	// application, password hashing would be in place.
	if err != nil || user.Password != password {
		// If we couldn't get the username from the DB, or if the password
		// doesn't match, re-run the login transaction with an error
		// message displayed.

		return sess.login, "Username or password not valid.", nil
	}

	// Login was successful. Update the session state to the logged-in user.
	sess.user = user

	return sess.mainmenu, nil, nil
}

var newuserScreen = go3270.Screen{
	{Row: 0, Col: 30, Intense: true, Content: "New User Registration"},
	{Row: 2, Col: 0,
		Content: "Please provide your user registration details."},

	// Username
	{Row: 4, Col: 0, Content: "Username . . .", Color: go3270.Green},
	{Row: 4, Col: 15, Name: loginUsername, Write: true,
		Highlighting: go3270.Underscore, Color: go3270.Turquoise},
	{Row: 4, Col: 24, Autoskip: true}, // field "stop" character

	// Password
	{Row: 5, Col: 0, Content: "Password . . .", Color: go3270.Green},
	{Row: 5, Col: 15, Name: loginPassword, Write: true, Hidden: true},
	{Row: 5, Col: 79, Autoskip: true}, // field "stop" character

	// Password
	{Row: 6, Col: 0, Content: "Confirm Pass .", Color: go3270.Green},
	{Row: 6, Col: 15, Name: loginPasswordConf, Write: true, Hidden: true},
	{Row: 6, Col: 79, Autoskip: true}, // field "stop" character

	// Name
	{Row: 7, Col: 0, Content: "Name . . . . .", Color: go3270.Green},
	{Row: 7, Col: 15, Name: loginName, Write: true,
		Highlighting: go3270.Underscore, Color: go3270.Turquoise},
	{Row: 7, Col: 46, Autoskip: true}, // field "stop" character

	// Error message
	{Row: 21, Col: 0, Name: loginErr, Color: go3270.Red, Intense: true},

	// Key legend
	{Row: 23, Col: 1, Content: "F1=Help"},
	{Row: 23, Col: 14, Content: "F3=Exit"},
}

var newuserScreenRules = go3270.Rules{
	loginUsername:     {Validator: go3270.NonBlank},
	loginPassword:     {Validator: go3270.NonBlank, Reset: true},
	loginPasswordConf: {Validator: go3270.NonBlank, Reset: true},
}

// newuserData is the data that can be passed into the newuser transaction
type newuserData struct {
	username string
	name     string
	errmsg   string
}

func (sess *session) newuser(conn net.Conn, _ go3270.DevInfo, data any) (
	go3270.Tx, any, error) {

	fieldValues := make(map[string]string)

	if data != nil {
		if newdata, ok := data.(newuserData); ok {
			fieldValues[loginUsername] = newdata.username
			fieldValues[loginName] = newdata.name
			fieldValues[loginErr] = newdata.errmsg
		}
	}

	resp, err := go3270.HandleScreen(
		newuserScreen,                 // the screen to display
		newuserScreenRules,            // the rules to enforce
		fieldValues,                   // pre-populated values in fields
		[]go3270.AID{go3270.AIDEnter}, // keys we accept -- validating
		[]go3270.AID{ // keys we accept -- non-validating
			go3270.AIDPF1,
			go3270.AIDPF3,
			go3270.AIDClear,
		},
		loginErr, // name of field to put error messages in
		4, 16,    // cursor coordinates
		conn)
	if err != nil {
		return nil, nil, err
	}

	switch resp.AID {
	case go3270.AIDClear:
		// Re-run transaction with empty values
		return sess.newuser, nil, nil
	case go3270.AIDPF1:
		// Display help screen, returning to this transaction after
		return help(sess.newuser), nil, nil
	case go3270.AIDPF3:
		// User wants to cancel registration, go back to login screen.
		return sess.login, nil, nil
	}

	// Otherwise, we'll try creating the user.

	username := resp.Values[loginUsername]
	password := resp.Values[loginPassword]
	passwordConf := resp.Values[loginPasswordConf]
	name := resp.Values[loginName]

	if password != passwordConf {
		// Re-run the newuser transaction with an error message
		return sess.newuser, newuserData{
			username: username,
			name:     name,
			errmsg:   "Passwords do not match.",
		}, nil
	}

	user, err := sess.db.CreateUser(User{
		Username: username,
		Password: password,
		Name:     name,
	})

	if err == ErrUserExists {
		// Re-run the newuser transaction with an error message
		return sess.newuser, newuserData{
			username: username,
			name:     name,
			errmsg:   "Username already exists; please choose a new one.",
		}, nil
	}

	if err != nil {
		return sess.newuser, newuserData{
			username: username,
			name:     name,
			errmsg:   "Unknown error creating new user.",
		}, nil
	}

	// Success! We'll stick the new user in the session state like login does
	// and go to the main menu.
	sess.user = user

	return sess.mainmenu, nil, nil
}
