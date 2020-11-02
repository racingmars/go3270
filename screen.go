// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

package go3270

import (
	"bytes"
	"fmt"
	"io"
)

// Field is a field on the 3270 screen.
type Field struct {
	// Row is the row, 0-based, that the field attribute character should
	// begin at. This library currently only supports 24 rows, so Row must
	// be 0-23.
	Row int

	// Col is the column, 0-based, that the field attribute character should
	// begin at. This library currently only supposed 80 columns, so Column
	// must be 0-79.
	Col int

	// Text is the content of the field to display.
	Content string

	// Write allows the user to edit the value of the field.
	Write bool

	// Intense indicates this field should be displayed with high intensity.
	Intense bool

	// Name is the name of this field, which is used to get the user-entered
	// data. All writeable fields on a screen must have a unique name.
	Name string
}

// Screen is an array of Fields which compose a complete 3270 screen.
// No checking is performed for lack of overlapping fields, unique field
// names,
type Screen []Field

// WriteScreen writes the 3270 datastream for the screen to a writer. Fields
// that aren't valid (e.g. outside of the 24x80 screen) are silently ignored.
// After writing the fields, the curser is set to crow, ccol, which are
// 0-based positions: row 0-23 and col 0-79. Errors from io.Writer.Write()
// are returned if encountered.
func WriteScreen(screen Screen, crow, ccol int, w io.Writer) error {
	var b bytes.Buffer

	b.WriteByte(0xf5) // Erase/Write to terminal
	b.WriteByte(0xc3) // WCC = Reset, Unlock Keyboard, Reset MDT

	// Build the commands for each field on the screen
	for _, fld := range screen {
		if fld.Row < 0 || fld.Row > 23 || fld.Col < 0 || fld.Col > 79 {
			// Invalid field position
			continue
		}

		b.Write(sba(fld.Row, fld.Col))
		b.Write(sf(fld.Write, fld.Intense))
		if fld.Content != "" {
			b.Write(a2e([]byte(fld.Content)))
		}
	}

	// Set cursor position. Correct out-of-bounds values to 0.
	if crow < 0 || crow > 23 {
		crow = 0
	}
	if ccol < 0 || ccol > 79 {
		ccol = 0
	}
	//b.Write(ic(crow, ccol))

	b.Write([]byte{0xff, 0xef}) // Telnet IAC EOR

	// Now write the datastream to the writer, returning any potential error.
	fmt.Printf("%x\n", b.Bytes())
	_, err := w.Write(b.Bytes())
	return err
}

// sba is the "set buffer address" 3270 command.
func sba(row, col int) []byte {
	result := make([]byte, 1, 3)
	result[0] = 0x11 // SBA
	result = append(result, getpos(row, col)...)
	return result
}

// sf is the "start field" 3270 command
func sf(write, intense bool) []byte {
	result := make([]byte, 2)
	result[0] = 0x1d // SF
	if !write {
		result[1] |= 1 << 5 // set "bit 2"
	}
	if intense {
		result[1] |= 1 << 3 // set "bit 4"
	}
	// Fill in top 2 bits with appropriate values
	result[1] = codes[result[1]]
	return result
}

// ic is the "insert cursor" 3270 command. This function will include the
// appropriate SBA command.
func ic(row, col int) []byte {
	result := make([]byte, 0, 3)
	result = append(result, sba(row, col)...)
	result = append(result, 0x13) // IC
	return result
}

// getpos translates row and col to buffer address control characters.
func getpos(row, col int) []byte {
	result := make([]byte, 2)
	address := row*80 + col
	hi := (address & 0xfc0) >> 6
	lo := address & 0x3f
	result[0] = codes[hi]
	result[1] = codes[lo]
	return result
}

// codes are the 3270 control character I/O codes, pre-computed as provided
// at http://www.tommysprinkle.com/mvs/P3270/iocodes.htm
var codes = []byte{0x40, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7, 0xc8,
	0xc9, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f, 0x50, 0xd1, 0xd2, 0xd3, 0xd4,
	0xd5, 0xd6, 0xd7, 0xd8, 0xd9, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f, 0x60,
	0x61, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7, 0xe8, 0xe9, 0x6a, 0x6b, 0x6c,
	0x6d, 0x6e, 0x6f, 0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7, 0xf8,
	0xf9, 0x7a, 0x7b, 0x7c, 0x7d, 0x7e, 0x7f}
