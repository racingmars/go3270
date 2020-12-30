// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

package go3270

import (
	"net"
	"time"
)

const (
	binary       = 0
	send         = 1
	se           = 240 // f0
	sb           = 250 // fa
	will         = 251 // fb
	wont         = 252 // fc
	do           = 253 // fd
	dont         = 254 // fe
	iac          = 255 // ff
	terminalType = 24  // 18
	eor          = 25  // 19
)

// NegotiateTelnet will naively (e.g. not checking client responses) negotiate
// the options necessary for tn3270 on a new telnet connection, conn.
func NegotiateTelnet(conn net.Conn) error {
	conn.Write([]byte{iac, do, terminalType})
	conn.Write([]byte{iac, sb, terminalType, send, iac, se})
	conn.Write([]byte{iac, do, eor})
	conn.Write([]byte{iac, do, binary})
	conn.Write([]byte{iac, will, eor, iac, will, binary})
	flushConnection(conn, time.Second*5)
	return nil
}

// UnNegotiateTelnet will naively (e.g. not checking client responses) attempt
// to restore the telnet options state to what it was before NegotiateTelnet()
// was called.
func UnNegotiateTelnet(conn net.Conn, timeout time.Duration) error {
	conn.Write([]byte{iac, wont, eor, iac, wont, binary})
	conn.Write([]byte{iac, dont, binary})
	conn.Write([]byte{iac, dont, eor})
	conn.Write([]byte{iac, dont, terminalType})
	flushConnection(conn, timeout)
	return nil
}

// flushConnection discards all bytes that it can read from conn, allowing up
// to the duration timeout for the first byte to be read.
func flushConnection(conn net.Conn, timeout time.Duration) error {
	defer conn.SetReadDeadline(time.Time{})
	buffer := make([]byte, 1024)
	for {
		conn.SetReadDeadline(time.Now().Add(timeout))
		n, err := conn.Read(buffer)
		if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
			debugf("nothing to flush\n")
			return nil
		}
		if err != nil {
			debugf("error while flushing: %v\n", err)
			return err
		}
		debugf("%d bytes read while flushing connection\n", n)
		// for follow-up reads, reduce the timeout
		timeout = time.Second / 2
	}
}

// telnetRead behaves similarly to conn.Read, except it filters out telnet
// commands. A single call to telnetRead() may perform multiple calls to
// c.Read() if awaiting completion of a command, or to ensure we do not return
// zero bytes in a non-error condition.
func telnetRead(c net.Conn, p []byte) (n int, err error) {
	const (
		normal = iota
		command
		subneg
	)

	// Not the most efficient, but for now to keep the code easier we'll just
	// read 1 byte at a time from the connection.
	b := make([]byte, 1)
	state := normal

	for n < len(p) {
		bn, berr := c.Read(b)

		// When there are no bytes to process and we received an error, we
		// are done no matter what state we're in. Any non-command bytes will
		// already be in p, so we return.
		if bn == 0 && berr != nil {
			return n, berr
		}

		// If we received 0 bytes but no error, we'll just read again.
		if bn == 0 {
			continue
		}

		// We got a byte! Let's progress through our state machine.
		switch state {
		case normal:
			if b[0] == iac {
				state = command
				debugf("entering telnet command state\n")
			} else {
				p[n] = b[0]
				n++
			}
		case command:
			if b[0] == 0xff {
				p[n] = 0xff
				n++
				state = normal
				debugf("leaving telnet command state; was an escaped 0xff\n")
			} else if b[0] == sb {
				state = subneg
				debugf("entering telnet command subnegotiation state\n")
			} else {
				state = normal
				debugf("leaving telnet command state; command was %02x\n",
					b[0])
			}
		case subneg:
			if b[0] == se {
				state = normal
				debugf("leaving telnet command subnegotiation state\n")
			} else {
				// remain in subnegotiation consuming bytes until we get se
				debugf("consumed telnet subnegotiation byte: %02x\n", b[0])
			}
		}

		// Now we handle any errors that returned partial data
		if berr != nil {
			return n, berr
		}
	}

	return n, nil
}
