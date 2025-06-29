// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

package go3270

import (
	"fmt"
	"io"
)

// Enable go3270 library debugging by setting Debug to an io.Writer.
// Disable debugging by setting it to nil (the default value).
var Debug io.Writer

// debugf will print to the Debug io.Writer if it isn't nil.
func debugf(format string, a ...interface{}) {
	if Debug == nil {
		return
	}

	fmt.Fprintf(Debug, "dbg: ")
	fmt.Fprintf(Debug, format, a...)
}

// codes are the 3270 control character I/O codes for 12-bit addressing,
// from Figure D-1 of GA23-0059-00. (Figure C-1 in later editions.)
var codes = []byte{0x40, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7, 0xc8,
	0xc9, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f, 0x50, 0xd1, 0xd2, 0xd3, 0xd4,
	0xd5, 0xd6, 0xd7, 0xd8, 0xd9, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f, 0x60,
	0x61, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7, 0xe8, 0xe9, 0x6a, 0x6b, 0x6c,
	0x6d, 0x6e, 0x6f, 0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7, 0xf8,
	0xf9, 0x7a, 0x7b, 0x7c, 0x7d, 0x7e, 0x7f}

// AIDtoString returns a string representation of an AID key name.
func AIDtoString(aid AID) string {
	switch aid {
	case AIDClear:
		return "Clear"
	case AIDEnter:
		return "Enter"
	case AIDNone:
		return "[none]"
	case AIDPA1:
		return "PA1"
	case AIDPA2:
		return "PA2"
	case AIDPA3:
		return "PA3"
	case AIDPF1:
		return "PF1"
	case AIDPF2:
		return "PF2"
	case AIDPF3:
		return "PF3"
	case AIDPF4:
		return "PF4"
	case AIDPF5:
		return "PF5"
	case AIDPF6:
		return "PF6"
	case AIDPF7:
		return "PF7"
	case AIDPF8:
		return "PF8"
	case AIDPF9:
		return "PF9"
	case AIDPF10:
		return "PF10"
	case AIDPF11:
		return "PF11"
	case AIDPF12:
		return "PF12"
	case AIDPF13:
		return "PF13"
	case AIDPF14:
		return "PF14"
	case AIDPF15:
		return "PF15"
	case AIDPF16:
		return "PF16"
	case AIDPF17:
		return "PF17"
	case AIDPF18:
		return "PF18"
	case AIDPF19:
		return "PF19"
	case AIDPF20:
		return "PF20"
	case AIDPF21:
		return "PF21"
	case AIDPF22:
		return "PF22"
	case AIDPF23:
		return "PF23"
	case AIDPF24:
		return "PF24"
	default:
		return "[unknown]"
	}
}
