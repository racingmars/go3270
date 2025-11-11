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

For an example of a complete application that handles multiple user sessions, adapts to different screen sizes, and with a nice written explanation of how the application is built, Moshix has create a [Minesweeper game][moshix-minesweeper] using go3270 to serve as a resource for the community to learn from.

[moshix-minesweeper]: https://github.com/moshix/minesweeper

Here's [a video introducing the library][introVideo] as well.

[introVideo]: https://www.youtube.com/watch?v=h9XTjup5W5U

A note on code pages
--------------------

After careful consideration, I have decided that the code page we will support for EBCDIC is IBM CP1047.

In suite3270 (e.g. c3270/x3270), the default code page is what it calls "brackets". This is CP37 with the [, ], Ý, and ¨ characters swapped around. This ends up placing all four of those characters in the correct place for 1047 (and thus they will work correctly with go3270 by default). However, the ^ and ¬ characters are swapped relative to CP1047. (Or, more succinctly, you could say the suite3270 "brackets" codepage is CP1047 with the ^ and ¬ characters swapped back to where they are in CP37). If you plan on using the ^ and ¬ characters, run c/x3270 in proper 1047 mode, `c3270 -codepage 1047` or make it your default by setting the `c3270.codePage` resource to `1047` in your `.c3270pro` file, for example.

In Vista TN3270, "United States" is the default code page. This is CP1047 and will map 100% correctly.

In IBM PCOMM, CP37 is the default. For correct mapping of [, ], Ý, ¨, ^, and ¬, you must switch the session parameters from "037 United States" to "1047 United States".

Additionally, most characters from the "graphic escape" code page 310 are supported. Correct display on the client will depend on its support of graphic escape and correct characters being available in its font. Use the corresponding Unicode characters in your Go UTF-8 strings and they will be sent as the EBCDIC two-byte sequence of 0x08 followed by the position in code page 310. GE sequences are also processed on incoming field values.

3270e mode
----------

Due to the design of the original API for this library (the lack of per-connection state information), it is not possible to support both tn3270 and tn3270e and auto-negotiate what the client supports. The library has traditionally used tn3270. It is possible to make the library globally operate in tn3270e mode by importing the `github.com/racingmars/go3270/tn3270e` package anywhere in your project, which has the side-effect of setting the global operating mode to tn3270e. There aren't any functional differences between the modes right now, except tn3270e mode might more successfully reject non-tn3270e clients at initial connection time.

For example, in your program's main.go, you would opt-in to global 3270e mode with:

```
package main

import (
    "github.com/racingmars/go3270"
    _ "github.com/racingmars/go3270/tn3270e"
)
```

tn3270e mode is currently experimental, please report any oddities you observe with it.

3270 information
----------------

I started learning about 3270 data streams from [Tommy Sprinkle's tutorial][sprinkle]. The tn3270 telnet negotiation is gleaned from [RFC 1576: TN3270 Current Practices][rfc1576], [RFC 1041: Telnet 3270 Regime Option][rfc1041], and [RFC 854: Telnet Protocol Specification][rfc854]. The IANA maintains a [useful reference of telnet option numbers][telnetOptions]. The reference I use for 3270 data streams is [the 1981 version from IBM][ibmref]. tn3270e is described in [RFC 2355: TN3270 Enhancements][rfc2355].

[sprinkle]: http://www.tommysprinkle.com/mvs/P3270/
[rfc1576]: https://tools.ietf.org/html/rfc1576
[rfc1041]: https://tools.ietf.org/html/rfc1041
[rfc854]: https://tools.ietf.org/html/rfc854
[telnetOptions]: https://www.iana.org/assignments/telnet-options/telnet-options.xhtml
[ibmref]: https://bitsavers.org/pdf/ibm/3270/GA23-0059-0_3270_Data_Stream_Programmers_Reference_Jan1981.pdf
[rfc2355]: https://tools.ietf.org/html/rfc2355

License
-------

This library is licensed under the MIT license; see the file LICENSE for details.
