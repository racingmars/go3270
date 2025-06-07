// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

package go3270

import (
	"bytes"
	"fmt"
	"net"
	"os"
)

// Response encapsulates data received from a 3270 client in response to the
// previously sent screen.
type Response struct {
	// Which Action ID key did the user press?
	AID AID

	// Row the cursor was on (0-based).
	Row int

	// Column the cursor was on (0-based).
	Col int

	// Field values.
	Values map[string]string
}

// AID is an Action ID character.
type AID byte

const (
	AIDNone  AID = 0x60
	AIDEnter AID = 0x7D
	AIDPF1   AID = 0xF1
	AIDPF2   AID = 0xF2
	AIDPF3   AID = 0xF3
	AIDPF4   AID = 0xF4
	AIDPF5   AID = 0xF5
	AIDPF6   AID = 0xF6
	AIDPF7   AID = 0xF7
	AIDPF8   AID = 0xF8
	AIDPF9   AID = 0xF9
	AIDPF10  AID = 0x7A
	AIDPF11  AID = 0x7B
	AIDPF12  AID = 0x7C
	AIDPF13  AID = 0xC1
	AIDPF14  AID = 0xC2
	AIDPF15  AID = 0xC3
	AIDPF16  AID = 0xC4
	AIDPF17  AID = 0xC5
	AIDPF18  AID = 0xC6
	AIDPF19  AID = 0xC7
	AIDPF20  AID = 0xC8
	AIDPF21  AID = 0xC9
	AIDPF22  AID = 0x4A
	AIDPF23  AID = 0x4B
	AIDPF24  AID = 0x4C
	AIDPA1   AID = 0x6C
	AIDPA2   AID = 0x6E
	AIDPA3   AID = 0x6B
	AIDClear AID = 0x6D
)

func readResponse(c net.Conn, fm fieldmap) (Response, error) {
	var r Response
	aid, err := readAID(c)
	if err != nil {
		return r, err
	}
	r.AID = aid

	// If the use pressed clear, or a PA key we should return now
	// TODO: actually, we should consume the 0xffef, but that will
	// currently get taken care of in our next AID search.
	if r.AID == AIDClear || r.AID == AIDPA1 || r.AID == AIDPA2 ||
		r.AID == AIDPA3 {
		return r, nil
	}

	row, col, _, err := readPosition(c)
	if err != nil {
		return r, err
	}
	r.Col = col
	r.Row = row

	var fieldValues map[string]string
	if fieldValues, err = readFields(c, fm); err != nil {
		return r, err
	}

	r.Values = fieldValues

	return r, nil
}

func readAID(c net.Conn) (AID, error) {
	for {
		b, valid, _, err := telnetRead(c, false)
		if !valid && err != nil {
			return AIDNone, err
		}
		if (b == 0x60) || (b >= 0x6b && b <= 0x6e) ||
			(b >= 0x7a && b <= 0x7d) || (b >= 0x4a && b <= 0x4c) ||
			(b >= 0xf1 && b <= 0xf9) || (b >= 0xc1 && b <= 0xc9) {
			// We found a valid AID
			debugf("Got AID byte: %x\n", b)
			return AID(b), nil
		}
		// Consume non-AID bytes continuing loop
		debugf("Got non-AID byte: %x\n", b)
	}
}

func readPosition(c net.Conn) (row, col, addr int, err error) {
	raw := make([]byte, 2)

	// Read two bytes
	for i := 0; i < 2; i++ {
		b, _, _, err := telnetRead(c, false)
		if err != nil {
			return 0, 0, 0, err
		}
		raw[i] = b
	}

	// Decode the raw position
	addr = decodeBufAddr([2]byte{raw[0], raw[1]})
	col = addr % 80
	row = (addr - col) / 80

	debugf("Got position bytes %02x %02x, decoded to %d\n", raw[0], raw[1],
		addr)

	return row, col, addr, nil
}

func readFields(c net.Conn, fm fieldmap) (map[string]string, error) {
	var infield bool
	var fieldpos int
	var fieldval bytes.Buffer
	var values = make(map[string]string)

	// consume bytes until we get 0xffef
	for {
		// Read a byte
		b, _, eor, err := telnetRead(c, true)
		if err != nil {
			return nil, err
		}

		// Check for end of data stream (0xffef)
		if eor {
			// Finish the current field
			if infield {
				debugf("Field %d: %s\n", fieldpos, e2a(fieldval.Bytes()))
				handleField(fieldpos, fieldval.Bytes(), fm, values)
			}

			return values, nil
		}

		// No? Check for start-of-field
		if b == 0x11 {
			// Finish the previous field, if necessary
			if infield {
				debugf("Field %d: %s\n", fieldpos, e2a(fieldval.Bytes()))
				handleField(fieldpos, fieldval.Bytes(), fm, values)
			}
			// Start a new field
			infield = true
			fieldval = bytes.Buffer{}
			fieldpos = 0

			if _, _, fieldpos, err = readPosition(c); err != nil {
				return nil, err
			}
			continue
		}

		// Consume all other bytes as field contents if we're in a field
		if !infield {
			debugf("Got unexpected byte while processing fields: %02x\n", b)
			continue
		}
		fieldval.WriteByte(b)
	}
}

func handleField(addr int, value []byte, fm fieldmap, values map[string]string) bool {
	name, ok := fm[addr]

	// Field is not present in the fieldmap
	if !ok {
		return false
	}

	// Otherwise, populate the value
	values[name] = string(e2a(value))
	return true
}

// decodeBufAddr decodes a raw 2-byte encoded buffer address and returns the
// integer value of the address (i.e. 0-1919)
func decodeBufAddr(raw [2]byte) int {
	if decodes[raw[0]] > 254 {
		fmt.Fprintf(os.Stderr,
			"UNEXPECTED VALUE: decodeBufAddr got raw value of %02x %02x\n",
			raw[0], raw[1])
	}
	if decodes[raw[1]] > 254 {
		fmt.Fprintf(os.Stderr,
			"UNEXPECTED VALUE: decodeBufAddr got raw value of %02x %02x\n",
			raw[0], raw[1])
	}

	hi := decodes[raw[0]] << 6
	lo := decodes[raw[1]]
	return hi | lo
}
