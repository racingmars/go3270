// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020, 2025 by Matthew R. Wilson, licensed under the MIT license.
// See LICENSE in the project root for license information.

package go3270

import (
	"bytes"
	"net"
	"strings"
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

	// Autoskip causes protected (Write = false) fields to automatically be
	// skipped and the cursor should move to the next field upon encountering
	// this field. Autoskip is ignored on fields with Write = true.
	Autoskip bool

	// Intense indicates this field should be displayed with high intensity.
	Intense bool

	// Hidden indicates the field content should not be displayed (e.g. a
	// password input field).
	Hidden bool

	// NumericOnly indicates that only numbers may be entered into the field.
	// Very fiew 3270 clients support this, so you must always still validate
	// the input on the server side.
	NumericOnly bool

	// Color is the field color. The default value is the default color.
	Color Color

	// Highlighting is the highlight attribute for the field. The default value
	// is the default (i.e. no) highlighting.
	Highlighting Highlight

	// Name is the name of this field, which is used to get the user-entered
	// data. All writeable fields on a screen must have a unique name.
	Name string

	// KeepSpaces will prevent the strings.TrimSpace() function from being
	// called on the field value. Generally you want leading and trailing
	// spaces trimmed from fields in 3270 before processing, but if you are
	// building a whitespace-sensitive application, you can ask for the
	// original, un-trimmed value for a field by setting this to true.
	KeepSpaces bool
}

// Color is a 3270 extended field attribute color value
type Color byte

// The valid 3270 colors
const (
	DefaultColor Color = 0
	Blue         Color = 0xf1
	Red          Color = 0xf2
	Pink         Color = 0xf3
	Green        Color = 0xf4
	Turquoise    Color = 0xf5
	Yellow       Color = 0xf6
	White        Color = 0xf7
)

// Highlight is a 3270 extended field attribute highlighting method
type Highlight byte

// The valid 3270 highlights
const (
	DefaultHighlight Highlight = 0
	Blink            Highlight = 0xf1
	ReverseVideo     Highlight = 0xf2
	Underscore       Highlight = 0xf4
)

// Screen is an array of Fields which compose a complete 3270 screen.
// No checking is performed for lack of overlapping fields, unique field
// names,
type Screen []Field

// ScreenOpts are the options that callers may set when sending a screen
// to the 3270 client.
type ScreenOpts struct {
	// NoResponse will draw the screen and immediately return, without
	// waiting for any input data from the remote client.
	NoResponse bool

	// NoClear will send the data stream to the remote client without
	// clearing the screen first. Existing data will be overlayed with
	// the current screen.
	NoClear bool

	// CursorRow sets the row (0-indexed) to position the cursor after
	// sending the screen, when NoClear is false. Maximum value is 23.
	CursorRow int

	// CursorCol sets the column (0-indexed) to position the cursor after
	// sending the screen, when NoClear is false. Maximum value is 79.
	CursorCol int
}

// fieldmap is a map of field buffer addresses and the corresponding field
// name.
type fieldmap map[int]string

// ShowScreenOpts writes the 3270 datastream for the screen, with the provided
// ScreenOpts, to a connection.
//
// Fields that aren't valid (e.g. outside of the 24x80 screen) are silently
// ignored. If a named field has an entry in the values map, the content of
// the field from the values map is used INSTEAD OF the Field struct's Content
// field. The values map may be nil if no overrides are needed.
//
// If opts.NoClear is false, the client screen will be cleared before writing
// the new screen, and the cursor will be repositioned to the values in
// opts.CursorRow and opts.CursorCol. If opts.NoClear is true, the screen will
// NOT be cleared, the cursor will NOT be repositioned, and the new screen
// will be overlayed over the current state of the client screen.
//
// If opts.NoResponse is false, ShowScreenOpts will block before returning,
// waiting for data from the client and returning the Response. If
// opts.NoResponse is true, ShowScreenOpts will immediately return after
// sending the datastream and the Response will be empty.
//
// If using from multiple threads -- one to block and wait for a response, and
// another to send screens with NoResponse and/or NoClear, be aware that if
// you change the input fields on screen after the initial blocking call is
// made, the response fields will not line up correctly and end up being
// invalid. That is to say, while waiting for a response, don't perform other
// actions from another thread that could layout the user input fields
// differently.
func ShowScreenOpts(screen Screen, values map[string]string, conn net.Conn,
	opts ScreenOpts) (Response, error) {

	var resp Response

	fm, err := showScreenInternal(screen, values, opts.CursorRow,
		opts.CursorCol, conn, !opts.NoClear)
	if err != nil {
		return resp, err
	}

	if !opts.NoResponse {
		resp, err = readResponse(conn, fm)
		if err != nil {
			return resp, err
		}

		// Strip spaces from field values unless the caller requested that we
		// maintain whitespace.
		for _, fld := range screen {
			if !fld.KeepSpaces {
				if _, ok := resp.Values[fld.Name]; ok {
					resp.Values[fld.Name] =
						strings.TrimSpace(resp.Values[fld.Name])
				}
			}
		}
	}

	return resp, nil
}

// Deprecated: use ShowScreenOpts with default/empty ScreenOpts.
func ShowScreen(screen Screen, values map[string]string, crow, ccol int,
	conn net.Conn) (Response, error) {

	return ShowScreenOpts(screen, values, conn,
		ScreenOpts{CursorRow: crow, CursorCol: ccol})
}

// Deprecated: use ShowScreenOpts with ScreenOpts.NoResponse = true.
func ShowScreenNoResponse(screen Screen, values map[string]string,
	crow, ccol int, conn net.Conn) error {

	_, err := ShowScreenOpts(screen, values, conn,
		ScreenOpts{NoResponse: true, CursorRow: crow, CursorCol: ccol})
	return err
}

func showScreenInternal(screen Screen, values map[string]string,
	crow, ccol int, conn net.Conn, clear bool) (fieldmap, error) {

	var b bytes.Buffer
	var fm = make(fieldmap) // field buffer positions -> name

	if clear {
		b.WriteByte(0xf5) // Erase/Write to terminal
	} else {
		b.WriteByte(0xf1) // Write to terminal
	}
	b.WriteByte(0xc3) // WCC = Reset, Unlock Keyboard, Reset MDT

	// Build the commands for each field on the screen
	for _, fld := range screen {
		if fld.Row < 0 || fld.Row > 23 || fld.Col < 0 || fld.Col > 79 {
			// Invalid field position
			continue
		}

		b.Write(sba(fld.Row, fld.Col))
		b.Write(buildField(fld))

		// Use fld.Content, unless the field is named and appears in the
		// value map.
		content := fld.Content
		if fld.Name != "" {
			if val, ok := values[fld.Name]; ok {
				content = val
			}
		}
		if content != "" {
			b.Write(a2e([]byte(content)))
		}

		// If a writable field, add it to the field map. We add 1 to bufaddr
		// to make the value match the reported position (I'm guessing it's
		// because we get the position of the field's first input position,
		// not the position of the field attribute byte).
		if fld.Write {
			bufaddr := fld.Row*80 + fld.Col
			fm[bufaddr+1] = fld.Name
		}
	}

	// If we cleared the screen, set the cursor position to the
	// caller-provided coordinates.
	if clear {
		// Set cursor position. Correct out-of-bounds values to 0.
		if crow < 0 || crow > 23 {
			crow = 0
		}
		if ccol < 0 || ccol > 79 {
			ccol = 0
		}
		b.Write(ic(crow, ccol))
	}

	b.Write([]byte{0xff, 0xef}) // Telnet IAC EOR

	// Now write the datastream to the writer, returning any potential error.
	debugf("sending datastream: %x\n", b.Bytes())
	if _, err := conn.Write(b.Bytes()); err != nil {
		return nil, err
	}

	return fm, nil
}

// sba is the "set buffer address" 3270 command.
func sba(row, col int) []byte {
	result := make([]byte, 1, 3)
	result[0] = 0x11 // SBA
	result = append(result, getpos(row, col)...)
	return result
}

// buildField will return either an sf or sfe command depending for the
// field.
func buildField(f Field) []byte {
	var buf bytes.Buffer
	if f.Color == DefaultColor && f.Highlighting == DefaultHighlight {
		// this is a traditional field, issue a normal sf command
		buf.WriteByte(0x1d) // sf - "start field"
		buf.WriteByte(sfAttribute(f.Write, f.Intense, f.Hidden, f.Autoskip,
			f.NumericOnly))
		return buf.Bytes()
	}

	// Otherwise, this needs an extended attribute field
	buf.WriteByte(0x29)     // sfe - "start field extended"
	var paramCount byte = 1 // we will always have the basic field attribute
	if f.Color != DefaultColor {
		paramCount++
	}
	if f.Highlighting != DefaultHighlight {
		paramCount++
	}
	buf.WriteByte(paramCount)

	// Write the basic field attribute
	buf.WriteByte(0xc0)
	buf.WriteByte(sfAttribute(f.Write, f.Intense, f.Hidden, f.Autoskip,
		f.NumericOnly))

	// Write the highlighting attribute
	if f.Highlighting != DefaultHighlight {
		buf.WriteByte(0x41)
		buf.WriteByte(byte(f.Highlighting))
	}

	// Write the color attribute
	if f.Color != DefaultColor {
		buf.WriteByte(0x42)
		buf.WriteByte(byte(f.Color))
	}

	return buf.Bytes()
}

// sfAttribute builds the attribute byte for the "start field" 3270 command
func sfAttribute(write, intense, hidden, skip, numeric bool) byte {
	var attribute byte
	if !write {
		attribute |= 1 << 5 // set "bit 2"
		if skip {
			attribute |= 1 << 4 // set "bit 3"
		}
	} else {
		// The MDT bit -- we always want writable field values returned,
		// even if unchanged
		attribute |= 1 // set "bit 7"
		if numeric {
			attribute |= 1 << 4 // set "bit 3"
		}
	}
	if intense {
		attribute |= 1 << 3 // set "bit 4"
	}
	if hidden {
		attribute |= 1 << 3 // set "bit 4"
		attribute |= 1 << 2 // set "bit 5"
	}
	// Fill in top 2 bits with appropriate values
	attribute = codes[attribute]
	return attribute
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
