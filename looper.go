// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

package go3270

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// Rules is a map of field names (strings) to FieldRules structs. Each field
// for which you wish validation to occur must appear in the map. Fields not
// in the map will not have any input validation performed.
type Rules map[string]FieldRules

// Validator is a type that represents a function which can perform field
// input validation. The function is passed a string, input, and returns
// true if the input is valid or false if the not.
type Validator func(input string) bool

// NonBlank is a Validator that returns true if, after spaces are trimmed from
// the beginning and end of the string, the value is not empty.
var NonBlank Validator = func(input string) bool {
	return !(strings.TrimSpace(input) == "")
}

var isIntegerRegexp = regexp.MustCompile(`^-?[0-9]+$`)

// IsInteger is a Validator that returns true if, after spaces are trimmed from
// the beginning and end if the string, the value is an integer (including
// negative numbers and 0).
var IsInteger Validator = func(input string) bool {
	input = strings.TrimSpace(input)
	return isIntegerRegexp.MatchString(input)
}

// FieldRules provides the validation rules for a particular field.
type FieldRules struct {
	// MustChange, when true, indicates that the value of the field MUST be
	// altered by the user -- if applied to a field with no starting value,
	// this makes the field a required field. If true on a field with a
	// starting value (either in the field's Content attribute, or with an
	// override in the initial values map), then the user must change
	// the value from the default.
	MustChange bool

	// ErrorText is the text displayed with the MustChange validation fails.
	// If ErrorText is the empty string, but MustValidation fails, an error
	// string will be constructed from the field name: "Please enter a valid
	// value for <fieldName>."
	ErrorText string

	// Validator is a function to validate the value the user input into the
	// field. It may be nil if no validation is required. The Validator
	// function is called *after* the MustChange logic, so if you wish to
	// fully handle validation, ensure MustChange is set to false.
	Validator Validator

	// Reset indicates that if the screen fails validation, this field should
	// always be reset to its original/default value, regardless of what the
	// user entered.
	Reset bool
}

// HandleScreen is a higher-level interface to the ShowScreen() function.
// HandleScreen will loop until all validation rules are satisfied, and only
// return when an expected AID (i.e. PF) key is pressed.
//
//   - screen is the Screen to display (see ShowScreen()).
//   - rules are the Rules to enforce: each key in the Rules map corresponds to
//     a Field.Name in the screen array.
//   - values are field values you wish to override (see ShowScreen()).
//   - pfkeys and exitkeys are the AID keys that you wish to accept (that is,
//     perform validation and return if successful) and treat as exit keys
//     (unconditionally return).
//   - errorField is the name of a field in the screen array that you wish error
//     messages to be written in when HandleScreen loops waiting for a valid
//     user submission.
//   - crow and ccol are the initial cursor position.
//   - conn is the network connection to the 3270 client.
//   - codepage is an optional argument (implemented this way as a varargs
//     argument as a hack to add this feature without breaking API backward
//     compatability) for the codepage to use. Typically you should pass in the
//     return value from DevInfo.Codepage() each time to get the correct
//     codepage that was detected when the client connected. If nil, the
//     global default code page (default 1047, but changed with the
//     SetCodepage() function) will be used.
//
// HandleScreen will return when the user: 1) presses a key in pfkeys AND all
// fields pass validation, OR 2) the user presses a key in exitkeys. In all
// other cases, HandleScreen will re-present the screen to the user again,
// possibly with an error message set in the errorField field.
//
// For alternate screen support (larger than 24x80), use HandleScreenAlt().
func HandleScreen(screen Screen, rules Rules, values map[string]string,
	pfkeys, exitkeys []AID, errorField string, crow, ccol int,
	conn net.Conn, codepage ...Codepage) (Response, error) {
	return HandleScreenAlt(screen, rules, values, pfkeys, exitkeys, errorField,
		crow, ccol, conn, nil, codepage...)
}

// HandleScreenAlt is identical to HandleScreen, but writes to the "alternate"
// screen size provided by dev. To write a non-24-by-80 screen, use this
// HandleScreenAlt function with a non-nil dev. If dev is nil, the behavior is
// identical to HandleScreen, which is limited to 24x80 and will set larger
// terminals to the default 24x80 mode.
func HandleScreenAlt(screen Screen, rules Rules, values map[string]string,
	pfkeys, exitkeys []AID, errorField string, crow, ccol int,
	conn net.Conn, dev DevInfo, codepage ...Codepage) (Response, error) {

	var cp Codepage
	if len(codepage) > 0 {
		cp = codepage[0]
	}

	// Save the original field values for any named fields to support
	// the MustChange rule. Also build a map of named fields.
	origValues := make(map[string]string)
	fields := make(map[string]*Field)
	for i := range screen {
		if screen[i].Name != "" {
			origValues[screen[i].Name] = screen[i].Content
			fields[screen[i].Name] = &screen[i]
		}
	}

	// Make our own field values map so we don't alter the caller's values
	myValues := make(map[string]string)
	for field := range values {
		myValues[field] = values[field]
	}

	// Now we loop...
mainloop:
	for {
		// Reset fields with FieldRules.Reset set
		for field := range rules {
			if rules[field].Reset {
				// avoid problems if there is a rule for a non-existent field
				if _, ok := fields[field]; ok {
					// Is the value in the origValues map?
					if value, ok := origValues[field]; ok {
						myValues[field] = value
					} else {
						// remove from the values map so we fall back to
						// whatever default is set for the field
						delete(myValues, field)
					}
				}
			}
		}

		resp, err := ShowScreenOpts(screen, myValues, conn,
			ScreenOpts{CursorRow: crow, CursorCol: ccol, AltScreen: dev,
				Codepage: cp})
		if err != nil {
			return resp, err
		}

		// If we got an exit key, return without performing validation
		if aidInArray(resp.AID, exitkeys) {
			return resp, nil
		}

		// If we got an unexpected key, set error message and restart loop
		if !aidInArray(resp.AID, pfkeys) {
			if !(resp.AID == AIDClear || resp.AID == AIDPA1 ||
				resp.AID == AIDPA2 || resp.AID == AIDPA3) {
				myValues = mergeFieldValues(myValues, resp.Values)
			}
			myValues[errorField] = fmt.Sprintf("%s: unknown key",
				AIDtoString(resp.AID))
			continue
		}

		// At this point, we have an expected key. If one of the "clear" keys
		// is expected, we can't do much, so we'll just return.
		if resp.AID == AIDClear || resp.AID == AIDPA1 || resp.AID == AIDPA2 ||
			resp.AID == AIDPA3 {
			return resp, nil
		}

		myValues = mergeFieldValues(myValues, resp.Values)
		delete(myValues, errorField) // don't persist errors across refreshes

		// Now we can validate each field
		for field := range rules {
			// skip rules for fields that don't exist
			if _, ok := myValues[field]; !ok {
				continue
			}
			if rules[field].MustChange &&
				myValues[field] == origValues[field] {
				myValues[errorField] = rules[field].ErrorText
				continue mainloop
			}
			if rules[field].Validator != nil &&
				!rules[field].Validator(myValues[field]) {
				myValues[errorField] = fmt.Sprintf(
					"Value for %s is not valid", field)
				continue mainloop
			}
		}

		// Everything passed validation
		return resp, nil
	}
}

// aidInArray performs a linear search through the aids array and returns true
// if aid appears in the array, false otherwise.
func aidInArray(aid AID, aids []AID) bool {
	for i := range aids {
		if aids[i] == aid {
			return true
		}
	}
	return false
}

// mergeFieldValues will return a new map, containing all keys from the current
// map and keys from the original map that do not exist in the current map.
// This is sometimes necessary because the caller of HandleScreen() may
// provide override values for non-writable fields, and we don't get those
// values back when we round-trip with the 3270 client.
func mergeFieldValues(original, current map[string]string) map[string]string {
	result := make(map[string]string)
	for key := range current {
		result[key] = current[key]
	}
	for key := range original {
		if _, ok := result[key]; !ok {
			result[key] = original[key]
		}
	}
	return result
}
