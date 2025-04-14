// This file is part of https://github.com/racingmars/go3270/
// Copyright 2025 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

// This example demonstrates using the RunTransactions() approach to
// structuring a go3270 application.

// This file contains a mock in-memory database for the purposes of our
// example application.

package main

import (
	"errors"
	"sync"
	"time"
)

type User struct {
	Username   string
	Password   string
	Name       string
	SignupDate time.Time
}

type DB interface {
	GetUser(username string) (User, error)
	CreateUser(user User) (User, error)
	UpdateUser(user User) (User, error)
}

type dbstate struct {
	lock  sync.Mutex
	users map[string]User
}

func Connect() DB {
	return &dbstate{
		users: make(map[string]User),
	}
}

var ErrUserExists = errors.New("username already exists")
var ErrNotFound = errors.New("record not found")

func (db *dbstate) GetUser(username string) (User, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	user, ok := db.users[username]
	if !ok {
		return User{}, ErrNotFound
	}

	return user, nil
}

func (db *dbstate) CreateUser(user User) (User, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	if _, ok := db.users[user.Username]; ok {
		return User{}, ErrUserExists
	}

	user.SignupDate = time.Now().UTC().Truncate(time.Second)

	// Obviously in a real application you wouldn't store plaintext password,
	// but this is a simple example for the purposes of demonstrating go3270,
	// not demonstrating how to build a secure application.
	db.users[user.Username] = user

	return user, nil
}

func (db *dbstate) UpdateUser(user User) (User, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	olduser, ok := db.users[user.Username]
	if !ok {
		return User{}, ErrNotFound
	}

	// Make sure original signup date is maintained
	user.SignupDate = olduser.SignupDate

	// Carry over existing password if we aren't setting a new password
	if user.Password == "" {
		user.Password = olduser.Password
	}

	db.users[user.Username] = user

	return user, nil
}
