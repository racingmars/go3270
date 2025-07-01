// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020, 2025 by Matthew R. Wilson, licensed under the MIT license.
// See LICENSE in the project root for license information.

package go3270

import (
	"errors"
	"net"
	"time"
)

// DevInfo provides information about the terminal that is connected.
//
// 3270 terminals operate at a default screen size of 24 rows that are 80
// columns wide. The normal "Write/Erase" datastream command always writes to
// the default 24x80 buffer. But some terminals support more rows and/or
// columns, and the alternate sized buffer may be written to with the
// "Write/Erase Alternate" command.
type DevInfo interface {
	// AltDimensions returns the number or rows and columns on the alternate
	// screen size.
	AltDimensions() (rows, cols int)

	// TerminalType reports the terminal-provided identification string. All
	// modern tn3270 clients will report one of the IBM-3278 models (-2, -3,
	// -4, or -5), or IBM-DYNAMIC if the alternate screen size isn't one of
	// the fixed sizes of the 3278 models. This string is purely
	// informational; the actual size of the alternate screen is available
	// from AltDimensions().
	TerminalType() string

	// Private version of AltDimensions() so callers can't fake us out; only
	// real implementations returned by NegotiateTelnet() will work.
	altDimensions() (rows, cols int)
}

const (
	se   = 240 // 0xf0
	sb   = 250 // 0xfa
	will = 251 // 0xfb
	wont = 252 // 0xfc
	do   = 253 // 0xfd
	dont = 254 // 0xfe
	iac  = 255 // 0xff

	// Options

	binaryOption = 0

	eorOption = 25  // 0x19
	eor       = 239 // 0xf1

	terminalType     = 24 // 0x18
	terminalTypeIs   = 0
	terminalTypeSend = 1
)

// ErrNo3270 indicates that the telnet client did not respond properly to the
// options negotiation that are expected for a tn3270 client.
var ErrNo3270 = errors.New("couldn't negotiate telnet options for tn3270")

// ErrTelnetError indicates an unexpected response was encountered in the
// telnet protocol.
var ErrTelnetError = errors.New("telnet or 3270 protocol error")

// ErrUnknownTerminal indicates the client did not identify itself as an
// IBM-3277, 3278, 3279, or IBM-DYNAMIC model. All modern tn3270 clients
// should report as IBM-3278 models or IBM-DYNAMIC.
var ErrUnknownTerminal = errors.New("unknown terminal type")

var errOptionRejected = errors.New("option rejected")

// NegotiateTelnet will negotiate the options necessary for tn3270 on a new
// telnet connection, conn.
func NegotiateTelnet(conn net.Conn) (DevInfo, error) {

	// Enable terminal type option
	if _, err := conn.Write([]byte{iac, do, terminalType}); err != nil {
		return nil, err
	}
	err := checkOptionResponse(conn, terminalType, do)
	if err == errOptionRejected || err == ErrTelnetError {
		return nil, ErrNo3270
	} else if err != nil {
		return nil, err
	}

	// Switch to the first available terminal type
	conn.Write([]byte{iac, sb, terminalType, terminalTypeSend, iac, se})
	devtype, err := getTerminalType(conn)
	if err == ErrTelnetError {
		return nil, ErrNo3270
	} else if err != nil {
		return nil, err
	}

	// Request end of record mode
	conn.Write([]byte{iac, do, eorOption})
	err = checkOptionResponse(conn, eorOption, do)
	if err == errOptionRejected || err == ErrTelnetError {
		return nil, ErrNo3270
	} else if err != nil {
		return nil, err
	}

	// Request binary mode
	conn.Write([]byte{iac, do, binaryOption})
	err = checkOptionResponse(conn, binaryOption, do)
	if err == errOptionRejected || err == ErrTelnetError {
		return nil, ErrNo3270
	} else if err != nil {
		return nil, err
	}

	// Enter end of record mode
	conn.Write([]byte{iac, will, eorOption})
	err = checkOptionResponse(conn, eorOption, will)
	if err == errOptionRejected || err == ErrTelnetError {
		return nil, ErrNo3270
	} else if err != nil {
		return nil, err
	}

	// Enter binary mode
	conn.Write([]byte{iac, will, binaryOption})
	err = checkOptionResponse(conn, binaryOption, will)
	if err == errOptionRejected || err == ErrTelnetError {
		return nil, ErrNo3270
	} else if err != nil {
		return nil, err
	}

	devinfo, err := makeDeviceInfo(conn, devtype)
	if err != nil {
		return nil, err
	}

	return devinfo, nil
}

// checkOptionResponse will check for the client's "will/wont" (if mode is do)
// or "do/dont" (if mode is will) response. mode is the option command the
// server just sent, and option is the option code to check for.
func checkOptionResponse(conn net.Conn, option, mode byte) error {
	var buf [3]byte

	var expectedYes, expectedNo byte
	switch mode {
	case do:
		expectedYes = will
		expectedNo = wont
	case will:
		expectedYes = do
		expectedNo = dont
	default:
		return ErrTelnetError
	}

	n, err := conn.Read(buf[:])
	if err != nil {
		return err
	}
	if n < 3 || buf[0] != iac {
		return ErrTelnetError
	}
	if buf[1] == expectedNo {
		// Was the correct option rejected?
		if buf[2] != option {
			return ErrTelnetError
		}
		return errOptionRejected
	}
	if buf[1] != expectedYes {
		return ErrTelnetError
	}

	// We have "will" now. But for the right option?
	if buf[2] != option {
		return ErrTelnetError
	}

	// All good, client accepted the option we requested.
	return nil
}

// getTerminalType reads the response to a "send terminal type" option
// subfield command.
func getTerminalType(conn net.Conn) (string, error) {
	var buf [100]byte
	var termtype string

	n, err := conn.Read(buf[:])
	if err != nil {
		return termtype, err
	}

	// At a minimum, with a one-character terminal type name, we expect
	// 7 bytes
	if n < 7 {
		return termtype, ErrTelnetError
	}

	// We'll check the expected control bytes all in one go...
	if buf[0] != iac || buf[1] != sb || buf[2] != terminalType ||
		buf[3] != terminalTypeIs || buf[n-2] != iac || buf[n-1] != se {
		return termtype, ErrTelnetError
	}

	// Everything looks good. The terminal type is an ASCII string between all
	// the control/command bytes.
	return string(buf[4 : n-2]), nil
}

func makeDeviceInfo(conn net.Conn, termtype string) (DevInfo, error) {
	// Known fixed size device types. All modern tn3270 clients should
	// report as 3278, but we'll also include 3277 and 3279 just in case.
	switch termtype {
	case "IBM-3277-2", "IBM-3277-2-E", "IBM-3278-2", "IBM-3278-2-E",
		"IBM-3279-2", "IBM-3279-2-E":
		return &deviceInfo{24, 80, termtype}, nil
	case "IBM-3278-3", "IBM-3278-3-E", "IBM-3279-3", "IBM-3279-3-E":
		return &deviceInfo{32, 80, termtype}, nil
	case "IBM-3278-4", "IBM-3278-4-E", "IBM-3279-4", "IBM-3279-4-E":
		return &deviceInfo{43, 80, termtype}, nil
	case "IBM-3278-5", "IBM-3278-5-E":
		return &deviceInfo{27, 132, termtype}, nil
	}

	// If it's not a fixed-size type, it should be IBM-DYNAMIC. If it isn't,
	// we don't know how to deal with it.
	if termtype != "IBM-DYNAMIC" {
		return nil, ErrUnknownTerminal
	}

	// For IBM-DYNAMIC, we need to discover the alternate screen size with
	// a structured field query.

	// First, we perform an ERASE / WRITE ALTERNATE to clear the screen
	// and put it in alternate screen mode. (EWA, reset WCC, telnet EOR)
	if _, err := conn.Write([]byte{0x7e, 0xc3, 0xff, 0xef}); err != nil {
		return nil, err
	}

	// Now we need to send the Write Structured Field command (0xf3) with the
	// "Read Partition - Query" structured field. Note that we're
	// telnet-escaping the 0xff in the data, but the subfield length is the
	// *unescaped* length (7).
	if _, err := conn.Write([]byte{0xf3, 0, 7, 0x01, 0xff, 0xff, 0x02,
		0xff, 0xef}); err != nil {
		return nil, err
	}

	var aid [1]byte
	n, err := conn.Read(aid[:])
	if err != nil {
		return nil, err
	}
	if n != 1 || aid[0] != byte(aidQueryResponse) {
		return nil, ErrTelnetError
	}

	var rows, cols int
	// There are an arbitrary number of query reply structured fields. We
	// are only interested in the "Usable Area" SFID=0x81 QCODE=0x81 field,
	// so we'll just consume any others. Consume all data until the EOR is
	// received.
	for {
		// Two bytes are big-endian length.
		buf, err := telnetReadN(conn, 2)
		if err != nil {
			return nil, err
		}
		if buf == nil {
			// EOR. We're out of fields.
			break
		}

		var l int = int(buf[0])<<8 + int(buf[1])

		// Field length includes the 2 length bytes
		buf, err = telnetReadN(conn, l-2)
		if err != nil {
			return nil, err
		}
		if buf == nil {
			return nil, ErrTelnetError
		}

		// Note that because length isn't at the beginning, offsets in buf
		// are 2 less than in the 3270 datastream documentation.

		if !(buf[0] == 0x81 && buf[1] == 0x81) {
			// Not 'Usable Area' query reply
			continue
		}

		// A valid Usable Area reply will always include at least 18 (20 with
		// length) bytes.
		if l < 18 {
			return nil, ErrTelnetError
		}

		// big-endian two byte values
		cols = int(buf[4])<<8 + int(buf[5])
		rows = int(buf[6])<<8 + int(buf[7])
	}

	if rows == 0 || cols == 0 {
		// We got an IBM-DYNAMIC device type, but it didn't include a
		// Usable Area query response.
		return nil, ErrUnknownTerminal
	}

	// We support 12- and 14-bit addressing. Using 16-bit addressing would
	// require a mode change and the current API design doesn't support
	// tracking the state necessary for that.
	//
	// We'll limit the reported screen size to what fits in 14-bit addressing
	// by removing rows if necessary.
	for rows*cols >= 1<<14 {
		rows--
	}

	return &deviceInfo{rows, cols, termtype}, nil
}

// UnNegotiateTelnet will naively (e.g. not checking client responses) attempt
// to restore the telnet options state to what it was before NegotiateTelnet()
// was called.
func UnNegotiateTelnet(conn net.Conn, timeout time.Duration) error {
	conn.Write([]byte{iac, wont, eorOption, iac, wont, binaryOption})
	conn.Write([]byte{iac, dont, binaryOption})
	conn.Write([]byte{iac, dont, eorOption})
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

// telnetReadN reads n unescaped, valid, non-EOR characters. The returned byte
// slice will always be length n (see special case below, though), unless
// error is non-nil, in which case the byte slice will be nil. Invalid or
// early EOR will return ErrTelnetError.
//
// AS A SPECIAL CASE, if the first byte read is EOR, then the returned byte
// slice AND error will be nil.
func telnetReadN(conn net.Conn, n int) ([]byte, error) {
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		b, valid, isEor, err := telnetRead(conn, true)
		if err != nil {
			return nil, err
		}
		if i == 0 && isEor {
			// If we're still on the first byte and it's EOR, return a
			// non-error nil value.
			return nil, nil
		}
		if !valid || isEor {
			return nil, ErrTelnetError
		}
		buf[i] = b
	}

	return buf, nil
}

type deviceInfo struct {
	rows, cols int
	termtype   string
}

func (d *deviceInfo) AltDimensions() (rows, cols int) {
	return d.rows, d.cols
}

func (d *deviceInfo) TerminalType() string {
	return d.termtype
}

func (d *deviceInfo) altDimensions() (rows, cols int) {
	return d.rows, d.cols
}
