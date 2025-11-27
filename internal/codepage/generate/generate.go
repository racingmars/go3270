// This file is part of https://github.com/racingmars/go3270/
// Copyright 2020, 2025 by Matthew R. Wilson, licensed under the MIT license.
// See LICENSE in the project root for license information.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Tool to generate codepage implementation files for go3270 using the
// Unicode icu-data UCM format files.

func main() {
	cpName := flag.String("n", "", "Code page name (e.g. 037)")
	cpPath := flag.String("i", "", "Input file path")
	flag.Parse()

	if *cpName == "" || *cpPath == "" {
		fmt.Fprintln(os.Stderr, "-n and -i arguments are required.")
		flag.Usage()
		os.Exit(1)
	}

	u2e, err := read(*cpPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// Build the reverse map
	e2u := make(map[int]int)
	for k, v := range u2e {
		e2u[v] = k
	}

	fmt.Println("// This file is part of https://github.com/racingmars/go3270/")
	fmt.Println("// Copyright 2020, 2025 by Matthew R. Wilson, licensed under the MIT license.")
	fmt.Println("// See LICENSE in the project root for license information.")
	fmt.Println()

	fmt.Println("package codepage")
	fmt.Println()
	fmt.Printf("// Codepage%s implements the IBM CP %s code page.\n",
		*cpName, *cpName)
	fmt.Println("//")
	fmt.Printf("// IBM CP %s <-> Unicode mappings from https://github.com/unicode-org/icu-data\n", *cpName)
	fmt.Printf("// file: %s\n", filepath.Base(*cpPath))
	fmt.Printf("var Codepage%s *codepage = &codepage{\n", *cpName)
	fmt.Printf("\tid: \"%s\",\n", *cpName)

	// Print 0x00-0xFF EBCDIC-to-Unicode
	line := 0
	pos := -1
	fmt.Printf("\te2u: []rune{\n")
	fmt.Printf("\t\t/*         x0    x1    x2    x3    x4    x5    x6    x7    x8    x9    xA    xB    xC    xD    xE    xF */\n")
	fmt.Printf("\t\t/* 0x */ ")
	for i := 0; i <= 0xFF; i++ {
		pos++
		if pos >= 16 {
			line++
			pos = 0
			fmt.Printf("\n")
			fmt.Printf("\t\t/* %Xx */ ", line)
		}
		v, ok := e2u[i]
		if !ok {
			fmt.Printf(" '�', ")
			continue
		}
		fmt.Printf("0x%02X, ", v)
	}
	fmt.Printf("\n\t},\n")

	// Print 0x00-0xFF Unicode-to-EBCDIC
	line = 0
	pos = -1
	fmt.Printf("\tu2e: []byte{\n")
	fmt.Printf("\t\t/*         x0    x1    x2    x3    x4    x5    x6    x7    x8    x9    xA    xB    xC    xD    xE    xF */\n")
	fmt.Printf("\t\t/* 0x */ ")
	for i := 0; i <= 0xFF; i++ {
		pos++
		if pos >= 16 {
			line++
			pos = 0
			fmt.Printf("\n")
			fmt.Printf("\t\t/* %Xx */ ", line)
		}
		v, ok := u2e[i]
		if !ok {
			fmt.Printf("0x3F, ")
			continue
		}
		fmt.Printf("0x%02X, ", v)
	}
	fmt.Printf("\n\t},\n")

	// Print Unicode-to-EBCDIC for codepoints >0xFF
	pos = -1
	fmt.Printf("\thighu2e: map[rune]byte{\n")
	fmt.Printf("\t\t")
	for k, v := range u2e {
		if k <= 0xFF {
			continue
		}
		pos++
		if pos >= 4 {
			pos = 0
			fmt.Printf("\n\t\t")
		}
		fmt.Printf("0x%04X: 0x%02X, ", k, v)
	}
	fmt.Printf("\n\t},\n")

	fmt.Println("\tesub:    0x3f,")
	fmt.Println("\tge:      0x08,")
	fmt.Println("\tge2u:    cp310ToUnicode,")
	fmt.Println("\tu2ge:    unicodeToCP310,")
	fmt.Println("}")
}

// read reads a UCM file and returns a map of Unicode CPs to EBCDIC
func read(input string) (map[int]int, error) {
	f, err := os.Open(input)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	u2e := make(map[int]int)

	s := bufio.NewScanner(f)

	var incharmap bool
	for s.Scan() {
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !incharmap && line != "CHARMAP" {
			continue
		}

		if line == "CHARMAP" {
			incharmap = true
			continue
		}

		// Skip non-roundtrip characters
		if strings.HasSuffix(line, "|1") {
			continue
		}

		if line == "END CHARMAP" {
			break
		}

		codepoint, ebcdic, err := parseUcmLine(line)
		if err != nil {
			panic(err)
		}

		if _, ok := u2e[codepoint]; ok {
			fmt.Fprintf(os.Stderr, "WARNING: duplicate codepoint U%04x\n",
				codepoint)
		}
		u2e[codepoint] = ebcdic
	}

	if err := s.Err(); err != nil {
		panic(err)
	}

	return u2e, nil
}

func parseUcmLine(s string) (int, int, error) {
	// Regex to match <UXXXX> and \xYY patterns
	reU := regexp.MustCompile(`U([0-9A-Fa-f]+)`)
	reX := regexp.MustCompile(`\\x([0-9A-Fa-f]+)`)

	// Find matches
	matchU := reU.FindStringSubmatch(s)
	matchX := reX.FindStringSubmatch(s)

	if matchU == nil || matchX == nil {
		return 0, 0, fmt.Errorf("could not find both hex patterns in input")
	}

	// Convert hex strings to integers
	valU, err := strconv.ParseInt(matchU[1], 16, 64)
	if err != nil {
		return 0, 0, err
	}

	valX, err := strconv.ParseInt(matchX[1], 16, 64)
	if err != nil {
		return 0, 0, err
	}

	return int(valU), int(valX), nil
}
