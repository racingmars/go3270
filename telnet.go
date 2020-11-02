// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

package go3270

import "io"

// NegotiateTelnet will naively (e.g. not checking client responses) negotiate
// the options necessary for tn3270 on a new telnet connection, conn.
func NegotiateTelnet(conn io.ReadWriter) error {
	rbuf := make([]byte, 255)

	conn.Write([]byte{0xff, 0xfd, 0x18}) // DO TermType
	conn.Read(rbuf)
	conn.Write([]byte{0xff, 0xfa, 0x18, 0x01, 0xff, 0xf0}) // TermType suboptions
	conn.Read(rbuf)
	conn.Write([]byte{0xff, 0xfd, 0x19}) // DO EOR
	conn.Read(rbuf)
	conn.Write([]byte{0xff, 0xfd, 0x00}) // DO Binary
	conn.Read(rbuf)

	conn.Write([]byte{0xff, 0xfb, 0x19, 0xff, 0xfb, 0x00}) // WILL binary, eor
	conn.Read(rbuf)

	return nil
}
