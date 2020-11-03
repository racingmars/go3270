// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

package go3270

import (
	"testing"
)

func TestEncode(t *testing.T) {
	encoded := getpos(0, 0)
	if encoded[0] != 0x40 || encoded[1] != 0x40 {
		t.Error("Position (0, 0) not correctly encoded")
	}

	encoded = getpos(11, 39)
	if encoded[0] != 0x4e || encoded[1] != 0xd7 {
		t.Error("Position (11, 39) not correctly encoded")
	}
}

func TestDecode(t *testing.T) {
	decoded := decodeBufAddr([2]byte{0x40, 0x40})
	if decoded != 0 {
		t.Error("Buffer address incorrectly decoded")
	}

	decoded = decodeBufAddr([2]byte{0x4e, 0xd7})
	if decoded != 919 {
		t.Error("Buffer address incorrectly decoded")
	}
}
