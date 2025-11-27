// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020, 2025 by Matthew R. Wilson, licensed under the MIT license.
// See LICENSE in the project root for license information.

package go3270

import (
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
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

	// Codepage is the Codepage interface that implements the EBCDIC
	// translation for the detected code page for the terminal, if supported.
	// This may be nil if the client code page is unknown. Whenever calling
	// the screen functions, always pass the value returned by this Codepage()
	// function in the ScreenOpts (nil is allowed to default to the global
	// default codepage).
	Codepage() Codepage

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

	// Sometimes the client will trigger us to send our "will" assertions
	// sooner than we otherwise would. Keep track here so we know not to send
	// them again.
	var sentWillBin, sentWillEOR bool

	// Enable terminal type option
	if _, err := conn.Write([]byte{iac, do, terminalType}); err != nil {
		return nil, err
	}
	err := checkOptionResponse(conn, terminalType, do,
		&sentWillEOR, &sentWillBin)
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
	err = checkOptionResponse(conn, eorOption, do,
		&sentWillEOR, &sentWillBin)
	if err == errOptionRejected || err == ErrTelnetError {
		return nil, ErrNo3270
	} else if err != nil {
		return nil, err
	}

	// Request binary mode
	conn.Write([]byte{iac, do, binaryOption})
	err = checkOptionResponse(conn, binaryOption, do,
		&sentWillEOR, &sentWillBin)
	if err == errOptionRejected || err == ErrTelnetError {
		return nil, ErrNo3270
	} else if err != nil {
		return nil, err
	}

	// It's possible there are already some client requests in the queue
	// that we haven't processed yet. We'll need to consume any outstanding
	// requests here and respond if necessary.
	var buf [3]byte
	for {
		conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		n, err := conn.Read(buf[:])
		conn.SetReadDeadline(time.Time{})
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				// No data waiting. We expect to eventually break out of the
				// for loop here.
				break
			} else {
				return nil, err
			}
		} else if n == 3 {
			if buf[0] == iac && buf[1] == do && buf[2] == eorOption {
				conn.Write([]byte{iac, will, eorOption})
				sentWillEOR = true
			} else if buf[0] == iac && buf[1] == do && buf[2] == binaryOption {
				conn.Write([]byte{iac, will, binaryOption})
				sentWillBin = true
			}
		} else {
			fmt.Println("SHORT READ SHORT READ")
		}
	}

	// Enter end of record mode
	if !sentWillEOR {
		conn.Write([]byte{iac, will, eorOption})
		err = checkOptionResponse(conn, eorOption, will,
			&sentWillEOR, &sentWillBin)
		if err == errOptionRejected || err == ErrTelnetError {
			return nil, ErrNo3270
		} else if err != nil {
			return nil, err
		}
	}

	// Enter binary mode
	if !sentWillBin {
		conn.Write([]byte{iac, will, binaryOption})
		err = checkOptionResponse(conn, binaryOption, will,
			&sentWillEOR, &sentWillBin)
		if err == errOptionRejected || err == ErrTelnetError {
			return nil, ErrNo3270
		} else if err != nil {
			return nil, err
		}
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
//
// If we end up getting a client request instead, we'll response and set
// sentEor or sentBin before trying to read the response again.
func checkOptionResponse(conn net.Conn, option, mode byte,
	sentEor, sentBin *bool) error {
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

	// If the client is requesting to negotiate a mode with us before the
	// response to our request, we'll satisfy it if it's one of the expected
	// modes and then try to read the client's response again.
	//
	// We only want to do this if we're not already expecting a "do" response
	// for the particular option.
	if !(expectedYes == do && buf[2] == option) {
		if buf[0] == iac && buf[1] == do && buf[2] == eorOption {
			conn.Write([]byte{iac, will, eorOption})
			*sentEor = true
			return checkOptionResponse(conn, option, mode, sentEor, sentBin)
		} else if buf[0] == iac && buf[1] == do && buf[2] == binaryOption {
			conn.Write([]byte{iac, will, binaryOption})
			*sentBin = true
			return checkOptionResponse(conn, option, mode, sentEor, sentBin)
		}
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

var modelRegex = regexp.MustCompile(`^IBM-\d{4}-([2-5])`)

func makeDeviceInfo(conn net.Conn, termtype string) (DevInfo, error) {
	var rows, cols, cpid int
	var codepage Codepage
	var isx3270 bool

	// tn3270e restricts to a small list of valid models, but since we're
	// not doing tn3270e, we are seeing a variety of model numbers. We'll
	// generically handle anything claiming to be a -2, -3, -4, or -5 type.
	//
	// We'll default to known terminal sizes in case we don't get the
	// structured field query response later.
	modelresult := modelRegex.FindStringSubmatch(termtype)
	if len(modelresult) == 2 {
		switch modelresult[1] {
		case "2":
			rows = 24
			cols = 80
		case "3":
			rows = 32
			cols = 80
		case "4":
			rows = 43
			cols = 80
		case "5":
			rows = 27
			cols = 132
		}
	} else if termtype != "IBM-DYNAMIC" {
		// If it's not a fixed-size type, it should be IBM-DYNAMIC. If it
		// isn't, we don't know how to deal with it. We'll just fall back on a
		// simple 24x80 assumption.
		rows = 24
		cols = 80
		termtype = "unknown (" + termtype + ")"
	}

	// Now we'll discover the terminal size and character set.

	// First, we perform an ERASE / WRITE ALTERNATE to clear the screen
	// and put it in alternate screen mode. (EWA, reset WCC, telnet EOR)
	if _, err := conn.Write([]byte{0x7e, 0xc3, 0xff, 0xef}); err != nil {
		return nil, err
	}

	// Now we need to send the Write Structured Field command (0xf3) with the
	// "Read Partition - Query" structured field. Note that we're
	// telnet-escaping the 0xff in the data, but the subfield length is the
	// *unescaped* length, including the 2 length bytes but excluding the
	// telnet EOR (5).
	if _, err := conn.Write([]byte{0xf3, 0, 5, 0x01, 0xff, 0xff, 0x02,
		0xff, 0xef}); err != nil {
		return nil, err
	}

	// We'll use a timeout in case the client doesn't support/reply to our
	// structured field query.
	var aid [1]byte
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := conn.Read(aid[:])
	conn.SetReadDeadline(time.Time{})
	if err != nil && errors.Is(err, os.ErrDeadlineExceeded) {
		// Timeout. In this case, we'll assume it's because the client didn't
		// reply to our query command. In that case, we'll return whatever
		// we're already assuming.
		return &deviceInfo{24, 80, termtype, nil}, nil
	} else if err != nil {
		return nil, err
	}
	if n != 1 || aid[0] != byte(aidQueryResponse) {
		return nil, ErrTelnetError
	}

	// There are an arbitrary number of query reply structured fields. We are
	// only interested in the "Usable Area" SFID=0x81 QCODE=0x81 field and
	// "Character Sets" QCODE=0x85 field so we'll just consume any others.
	// Consume all data until the EOR is received.
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

		// Note that because length isn't at the beginning, offsets in buf are
		// 2 less than in the 3270 data stream documentation.
		if buf[0] == 0x81 && buf[1] == 0x81 {
			// Usable Area
			rows, cols, err = getUsableArea(buf)
			if err != nil {
				return nil, err
			}
		} else if buf[0] == 0x81 && buf[1] == 0x85 {
			// Character Sets
			cpid = getCodepageID(buf)
		} else if buf[0] == 0x81 && buf[1] == 0xA1 {
			// RPQ Names. We use this  to determine if the client is x3270
			// family.
			isx3270 = getRPQNames(buf)
		} else {
			// Not a field we're interested in
			continue
		}
	}

	switch cpid {
	case 37:
		// If x3270 family, assume that this is really the default "bracket"
		// codepage, which reports as 37, not true CP37.
		if isx3270 {
			codepage = CodepageBracket()
		} else {
			codepage = Codepage037()
		}
	case 924:
		codepage = Codepage924()
	case 1047:
		codepage = Codepage1047()
	case 1140:
		codepage = Codepage1140()
	default:
		// nil codepage will be accepted in ScreenOpts to default to the
		// global default codepage.
		codepage = nil
	}

	return &deviceInfo{rows, cols, termtype, codepage}, nil
}

// getUsableArea processes the "Query Reply (Usable Area)" response to return
// the rows and columns count of the terminal. The byte slice passed in to buf
// must begin with {0x81, 0x81}.
func getUsableArea(buf []byte) (rows, cols int, err error) {
	// A valid Usable Area reply will always include at least 18 (20 with
	// length) bytes.
	if len(buf) < 18 || buf[0] != 0x81 || buf[1] != 0x81 {
		return 0, 0, ErrTelnetError
	}

	// big-endian two byte values
	cols = int(buf[4])<<8 + int(buf[5])
	rows = int(buf[6])<<8 + int(buf[7])

	if rows == 0 || cols == 0 {
		// Got a Usable Area response but the values are 0?
		return 0, 0, ErrUnknownTerminal
	}

	// We support 12- and 14-bit addressing. Using 16-bit addressing
	// would require a mode change and the current API design doesn't
	// support tracking the state necessary for that.
	//
	// We'll limit the reported screen size to what fits in 14-bit
	// addressing by removing rows if necessary.
	for rows*cols >= 1<<14 {
		rows--
	}

	return rows, cols, nil
}

// getCodepageID processes the "Query Reply (Character Sets)" response to
// return the integer code page number if present. If unable, returns 0. The
// byte slice passed in to buf must begin with {0x81, 0x85}.
func getCodepageID(buf []byte) int {
	// Initial validity check.
	if len(buf) < 11 || buf[0] != 0x81 || buf[1] != 0x85 {
		return 0
	}

	// If the GF bit is not set, no point in continuing.
	if buf[2]&(1<<1) != 1<<1 {
		return 0
	}

	// Descriptor length
	dl := int(buf[10])

	// There may be more than one descriptors, and we need to find the first
	// one with local ID 0.
	pos := 11 // first descriptor
	for {
		if len(buf) < pos+dl {
			// No more descriptors and we haven't found anything yet
			return 0
		}

		if buf[pos] != 0 {
			// not the descriptor we're looking for, try the next one
			pos += dl
			continue
		}

		// This is the first descriptor we've seen with ID 0, we'll use it.
		// No matter how long the descriptor is, the code page will 2-byte big
		// endian integer in the last two bytes.
		cpid := int(buf[pos+dl-2])<<8 + int(buf[pos+dl-1])
		return cpid
	}
}

// getRPGNames checks the "Query Reply (RPQ NAMES)" response to see if the
// client is in the x3270 family. The byte slice passed in to buf must begin
// with {0x81, 0xA1}.
func getRPQNames(buf []byte) bool {
	if len(buf) < 16 {
		return false
	}

	// "x3270" in EBCDIC
	if buf[11] == 0xa7 && buf[12] == 0xf3 && buf[13] == 0xf2 &&
		buf[14] == 0xf7 && buf[15] == 0xf0 {
		return true
	}

	return false
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
	codepage   Codepage
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

func (d *deviceInfo) Codepage() Codepage {
	return d.codepage
}
