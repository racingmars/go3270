// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020 by Matthew R. Wilson, licensed under the MIT license. See
// LICENSE in the project root for license information.

package go3270

import "github.com/racingmars/go3270/internal/codepage"

// Implementations of Codepage provide EBCDIC<->UTF-8 translation. By default,
// go3270 is configured to use CP 1047. You may alternatively set a different
// codepage using the SetCodepage() function during your application
// initialization.
type Codepage interface {
	// Decode converts a slice of EBCDIC bytes into a UTF-8 string.
	Decode(e []byte) string

	// Encode converts a UTF-8 string into a slice of EBCDIC bytes.
	Encode(s string) []byte

	// ID returns the name of this codepage. Usually a numeric string like
	// "037" or "1047", but could also be a name such as "bracket" if IBM has
	// not assigned a number to the particular codepage.
	ID() string
}

// After careful consideration, I have decided that the default code page we
// will support for EBCDIC is IBM CP 1047. Other code pages may be globally
// selected with the SetCodepage() function.
//
// In suite3270 (e.g. c3270/x3270), the default code page is what it calls
// "brackets". This is CP37 with the [, ], Ý, and ¨ characters swapped around.
// This ends up placing all four of those characters in the correct place for
// 1047 (and thus they will all work correctly with go3270 by default).
// HOWEVER, the ^ and ¬ characters are swapped relative to CP1047. (Or, more
// succinctly, you could say the suite3270 "brackets" codepage is CP1047 with
// the ^ and ¬ characters swapped back to where they are in CP37). If you plan
// on using the ^ and ¬ characters, run c/x3270 in proper 1047 mode,
// `c3270-codepage 1047` or make it your default by setting the
// `c3270.codePage` resource to `1047` in your `.c3270pro` file, for example.
//
// In Vista TN3270, "United States" is the default code page. This is CP1047
// and will map 100% correctly.
//
// In IBM PCOMM, CP37 is the default. For correct mapping of [, ], Ý, ¨, ^,
// and ¬, you must switch the session parameters from "037 United States" to
// "1047 United States".
var defaultCodepage Codepage = Codepage1047()

// SetCodepage sets the codepage/character set that go3270 uses. This is a
// global setting, so if you're expecting clients to be configured to use a
// character set other than go3270's default, cp1047, you should probably set
// this during your application initialization and then leave it unchanged
// after. This is _not_ a per-connection setting.
//
// For per-client codepage, set the ScreenOpts.Codepage field in the calls to
// ShowScreenOpts() or the codepage argument to HandleScreen() and
// HandleScreenAlt().
func SetCodepage(cs Codepage) {
	defaultCodepage = cs
}

func CodepageBracket() Codepage { return codepage.CodepageBracket }
func Codepage037() Codepage     { return codepage.Codepage037 }
func Codepage273() Codepage     { return codepage.Codepage273 }
func Codepage275() Codepage     { return codepage.Codepage275 }
func Codepage277() Codepage     { return codepage.Codepage277 }
func Codepage278() Codepage     { return codepage.Codepage278 }
func Codepage280() Codepage     { return codepage.Codepage280 }
func Codepage284() Codepage     { return codepage.Codepage284 }
func Codepage285() Codepage     { return codepage.Codepage285 }
func Codepage297() Codepage     { return codepage.Codepage297 }
func Codepage424() Codepage     { return codepage.Codepage424 }
func Codepage500() Codepage     { return codepage.Codepage500 }
func Codepage803() Codepage     { return codepage.Codepage803 }
func Codepage870() Codepage     { return codepage.Codepage870 }
func Codepage871() Codepage     { return codepage.Codepage871 }
func Codepage875() Codepage     { return codepage.Codepage875 }
func Codepage880() Codepage     { return codepage.Codepage880 }
func Codepage924() Codepage     { return codepage.Codepage924 }
func Codepage1026() Codepage    { return codepage.Codepage1026 }
func Codepage1047() Codepage    { return codepage.Codepage1047 }
func Codepage1140() Codepage    { return codepage.Codepage1140 }
func Codepage1141() Codepage    { return codepage.Codepage1141 }
func Codepage1142() Codepage    { return codepage.Codepage1142 }
func Codepage1143() Codepage    { return codepage.Codepage1143 }
func Codepage1144() Codepage    { return codepage.Codepage1144 }
func Codepage1145() Codepage    { return codepage.Codepage1145 }
func Codepage1146() Codepage    { return codepage.Codepage1146 }
func Codepage1147() Codepage    { return codepage.Codepage1147 }
func Codepage1148() Codepage    { return codepage.Codepage1148 }
func Codepage1149() Codepage    { return codepage.Codepage1149 }
func Codepage1160() Codepage    { return codepage.Codepage1160 }

var codepageToFunction map[int]func() Codepage = map[int]func() Codepage{
	37:   Codepage037,
	273:  Codepage273,
	275:  Codepage275,
	277:  Codepage277,
	278:  Codepage278,
	280:  Codepage280,
	284:  Codepage284,
	285:  Codepage285,
	297:  Codepage297,
	424:  Codepage424,
	500:  Codepage500,
	803:  Codepage803,
	870:  Codepage870,
	871:  Codepage871,
	875:  Codepage875,
	880:  Codepage880,
	924:  Codepage924,
	1026: Codepage1026,
	1047: Codepage1047,
	1140: Codepage1140,
	1141: Codepage1141,
	1142: Codepage1142,
	1143: Codepage1143,
	1144: Codepage1144,
	1145: Codepage1145,
	1146: Codepage1146,
	1147: Codepage1147,
	1148: Codepage1148,
	1149: Codepage1149,
	1160: Codepage1160,
}
