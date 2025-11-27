Go 3270 Server Library
======================

[![PkgGoDev](https://pkg.go.dev/badge/github.com/racingmars/go3270)](https://pkg.go.dev/github.com/racingmars/go3270)

This library allows you to write Go servers for tn3270 clients by building 3270 data streams from fields and processing the client's response to receive the attention keys and field values entered by users.

**Project status:** This library has been used by a small number of projects, and I believe the overall functionality is sound and relatively bug-free. Feedback is appreciated. At this point, I will try not to make breaking changes to the API, but I have not yet declared the library to be at v1.0, so I make no promises.

Usage
-----

See the example folders for quick demonstrations of using the library:

 * example1 uses the lower-level function ShowScreenOpts().
 * example2 uses a higher-level convenience function HandleScreen().
 * example3 demonstrates updating the client's 3270 display while waiting for a response by using an update thread and a waiting response thread.
 * example4 demonstrates the RunTransactions() approach to handing control from one screen to the next. This is the recommended way to build applications using go3270.
 * example5 demonstrates support for larger-than-default (24x80) terminal sizes.

For larger applications, I recommend using the `RunTransactions()` function to serve as the driver for your application. You can implement transaction functions which pass control from one transaction to another. example4 demonstrates a "larger" application that uses this approach.

For an example of a complete application that handles multiple user sessions, adapts to different screen sizes, and with a nice written explanation of how the application is built, Moshix has created a [Minesweeper game][moshix-minesweeper] using go3270 to serve as a resource for the community to learn from.

[moshix-minesweeper]: https://github.com/moshix/minesweeper

Here's [a video introducing the library][introVideo] as well.

[introVideo]: https://www.youtube.com/watch?v=h9XTjup5W5U

Code page support
-----------------

When clients connect, the `NegotiateTelnet()` function returns an implementation of the `DevInfo` interface. That interface has a `DevInfo.Codepage()` method, which returns an implementation of the `Codepage` interface for the detected client code page if known and supported (`nil` otherwise). The `ScreenOpts` structure you provide to the `ShowScreenOpts()` function has a `Codepage` field, which is where you provide `DevInfo.Codepage()` to ensure the UTF-8–EBCDIC translation is correct for the particular client. If `ScreenOpts.Codepage` is `nil` (either because you don't set it, or if `DevInfo.Codepage()` returns `nil`), then go3270 will use its global default code page for that interaction with the client. The global default is the CP1047 code page.

All of the examples under this repository demonstrate the correct handling of code pages -- the `DevInfo` is remembered from the telnet negotiation, and `DevInfo.Codepage()` is passed to all screen send and receive calls.

You may change the global code page default by calling the SetCodepage() function during your application initialization (this should be set before you use the library for handling any client connections; this is a global setting, not a per-connection setting). SetCodepage() accepts a Codepage interface, which provides methods to encode Go UTF-8 strings to EBCDIC, and decode EBCDIC byte slices to Go UTF-8 strings.

go3270 currently provides the following codepage functions, covering all single-byte code pages supported by x3270 v4.3ga10:

 * `Codepage037()`
 * `Codepage273()`
 * `Codepage275()`
 * `Codepage277()`
 * `Codepage278()`
 * `Codepage280()`
 * `Codepage284()`
 * `Codepage285()`
 * `Codepage297()`
 * `Codepage424()`
 * `Codepage500()`
 * `Codepage803()`
 * `Codepage870()`
 * `Codepage871()`
 * `Codepage875()`
 * `Codepage880()`
 * `Codepage924()`
 * `Codepage1026()`
 * `Codepage1047()`
 * `Codepage1140()`
 * `Codepage1141()`
 * `Codepage1142()`
 * `Codepage1143()`
 * `Codepage1144()`
 * `Codepage1145()`
 * `Codepage1146()`
 * `Codepage1147()`
 * `Codepage1148()`
 * `Codepage1149()`
 * `Codepage1160()`

Additionally, the default x3270 family code page (closest to CP 1047, with with `^` and `¬` swapped back to where they are in CP 37), `CodepageBracket()`. (Note that x3270 does not report a different code page ID between CP37 and "bracket". Since "bracket" is the default code page for x3270, go3270 will assume that if the client is in the x3270 family and the codepage is 37, "bracket" is the correct code page to use.)

If there are other standard EBCDIC code pages that you would like support for, let me know.

To configure go3270 to use one of the codepages by default, you may do something like:

```
import (
    "github.com/racingmars/go3270"
)

func init() {
    go3270.SetCodepage(go3270.Codepage1047())
}
```

With the new codepage detection support, you shouldn't typically rely on the default codepage: you should always pass the `DevInfo.Codepage()` value to the `ShowScreenOpts()` function or as the last optional argument to the `HandleScreen()` or `HandleScreenAlt()` functions. The global default should only be a fallback if the client code page isn't detected correctly.

Additionally, most characters from the "graphic escape" code page 310 are supported in all of the go3270-provided codepage implementations. Correct display on the client will depend on its support of graphic escape and correct characters being available in its font. Use the corresponding Unicode characters in your Go UTF-8 strings and they will be sent as the EBCDIC two-byte sequence of 0x08 followed by the position in code page 310. GE sequences are also processed on incoming field values.

3270 information
----------------

I started learning about 3270 data streams from [Tommy Sprinkle's tutorial][sprinkle]. The tn3270 telnet negotiation is gleaned from [RFC 1576: TN3270 Current Practices][rfc1576], [RFC 1041: Telnet 3270 Regime Option][rfc1041], and [RFC 854: Telnet Protocol Specification][rfc854]. The IANA maintains a [useful reference of telnet option numbers][telnetOptions]. The reference I use for 3270 data streams is [the 1981 version from IBM][ibmref].

[sprinkle]: http://www.tommysprinkle.com/mvs/P3270/
[rfc1576]: https://tools.ietf.org/html/rfc1576
[rfc1041]: https://tools.ietf.org/html/rfc1041
[rfc854]: https://tools.ietf.org/html/rfc854
[telnetOptions]: https://www.iana.org/assignments/telnet-options/telnet-options.xhtml
[ibmref]: https://bitsavers.org/pdf/ibm/3270/GA23-0059-0_3270_Data_Stream_Programmers_Reference_Jan1981.pdf

License
-------

This library is licensed under the MIT license; see the file LICENSE for details.
