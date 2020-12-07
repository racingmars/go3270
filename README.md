Go 3270 Server Library
======================

[![PkgGoDev](https://pkg.go.dev/badge/github.com/racingmars/go3270)](https://pkg.go.dev/github.com/racingmars/go3270)

This library allows you to write Go servers for tn3270 clients by building 3270 data streams from fields and processing the client's response to receive the attention keys and field values entered by users.

**The library is incomplete, likely buggy, and under heavy development: the interface is UNSTABLE until this notice is removed from this readme and version 1.0 is released.**

Everything I know about 3270 data streams I learned from [Tommy Sprinkle's tutorial][sprinkle]. The tn3270 telnet negotiation is gleaned from [RFC 1576: TN3270 Current Practices][rfc1576], [RFC 1041: Telnet 3270 Regime Option][rfc1041], and [RFC 854: Telnet Protocol Specification][rfc854].

[sprinkle]: http://www.tommysprinkle.com/mvs/P3270/
[rfc1576]: https://tools.ietf.org/html/rfc1576
[rfc1041]: https://tools.ietf.org/html/rfc1041
[rfc854]: https://tools.ietf.org/html/rfc854

Usage
-----

See the example folders for a quick demonstration of using the library. The examples are very similar, but example1 uses the lower-level function ShowScreen(), and example2 uses a higher-level function HandleScreen().

Here's [a video introducing the library][introVideo] as well.

[introVideo]: https://www.youtube.com/watch?v=h9XTjup5W5U

Future Enhancements
-------------------

I would like to add:

 - ~~Extended field attribute support (e.g. color).~~ **Done**
 - Utility functions for easily laying out forms.

Known Problems
--------------

 - The telnet data is not checked for the special telnet byte value, 0xFF, which requires escaping while sending and unescaping while receiving data. If your 3270 data streams contain an FF character, things will probably break. However...I don't think 0xFF should ever appear in the datastream, so we're probably okay.
 - The telnet negotiation does not check for any errors or for any responses from the client. We just assume it goes well and we're actually talking to a tn3270 client.

License
-------

This library is licensed under the MIT license; see the file LICENSE for details.
