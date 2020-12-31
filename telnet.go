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
	eoroption    = 25  // 19
	eor          = 239 // f1
)

// NegotiateTelnet will naively (e.g. not checking client responses) negotiate
// the options necessary for tn3270 on a new telnet connection, conn.
func NegotiateTelnet(conn net.Conn) error {
	conn.Write([]byte{iac, do, terminalType})
	conn.Write([]byte{iac, sb, terminalType, send, iac, se})
	conn.Write([]byte{iac, do, eoroption})
	conn.Write([]byte{iac, do, binary})
	conn.Write([]byte{iac, will, eoroption, iac, will, binary})
	flushConnection(conn, time.Second*5)
	return nil
}

// UnNegotiateTelnet will naively (e.g. not checking client responses) attempt
// to restore the telnet options state to what it was before NegotiateTelnet()
// was called.
func UnNegotiateTelnet(conn net.Conn, timeout time.Duration) error {
	conn.Write([]byte{iac, wont, eoroption, iac, wont, binary})
	conn.Write([]byte{iac, dont, binary})
	conn.Write([]byte{iac, dont, eoroption})
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

// telnetRead returns the next byte of data from the connection c, but
// filters out all telnet commands. If passEOR is true, then telnetRead will
// return upon encountering the telnet End of Record command, setting isEor to
// true. When isEor is true, the value of b is meaningless and must be ignored
// (valid will be false). When valid is true, the value in byte b is a real
// value read from the connection; when value is false, do not use the value
// in b. (For example, a valid byte AND error can be returned in the same
// call.)
func telnetRead(c net.Conn, passEOR bool) (b byte, valid, isEor bool, err error) {
	const (
		normal = iota
		command
		subneg
	)

	buf := make([]byte, 1)
	state := normal

	for {
		bn, berr := c.Read(buf)

		// When there are no bytes to process and we received an error, we
		// are done no matter what state we're in. Any non-command bytes will
		// already be in p, so we return.
		if bn == 0 && berr != nil {
			return 0, false, false, berr
		}

		// If we received 0 bytes but no error, we'll just read again.
		if bn == 0 {
			continue
		}

		// We got a byte! Let's progress through our state machine.
		switch state {
		case normal:
			if buf[0] == iac {
				state = command
				debugf("entering telnet command state\n")
			} else {
				return buf[0], true, false, berr
			}
		case command:
			if buf[0] == 0xff {
				debugf("leaving telnet command state; was an escaped 0xff\n")
				return 0xff, true, false, nil
			} else if buf[0] == sb {
				state = subneg
				debugf("entering telnet command subnegotiation state\n")
			} else if passEOR && buf[0] == eor {
				debugf("leaving telnet command state; returning EOR\n")
				return 0, false, true, nil
			} else {
				state = normal
				debugf("leaving telnet command state; command was %02x\n",
					buf[0])
			}
		case subneg:
			if buf[0] == se {
				state = normal
				debugf("leaving telnet command subnegotiation state\n")
			} else {
				// remain in subnegotiation consuming bytes until we get se
				debugf("consumed telnet subnegotiation byte: %02x\n", buf[0])
			}
		}
	}
}
