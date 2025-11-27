package codepage

import "unicode/utf8"

// Internal implementation of the Charset interface we'll use for the codepage
// support we provide.
type codepage struct {
	// EBCDIC byte to Unicode code point for bytes 0x00-0xFF
	e2u []rune

	// Unicode code point to EBCDIC byte for codepoints 0x00-0xFF
	u2e []byte

	// Map of Unicode code points to EBCDIC bytes for codepoints >0xFF
	highu2e map[rune]byte

	// The EBCDIC substitute character to use if there is no EBCDIC character
	// for the requested Unicode code point (typically 0x3F).
	esub byte

	// The "graphic escape" EBCDIC byte (is it ever anything other than 0x0E?)
	ge byte

	// Graphic escape codepage EBCDIC byte to Unicode code point for bytes
	// 0x00-0xFF. Use rune '�' for unmapped bytes.
	ge2u []rune

	// Map of Unicode code points to graphic escape EBCDIC bytes.
	u2ge map[rune]byte

	id string
}

// Decode will convert an EBCDIC byte array into a UTF-8 Go string, handling
// graphic escape to CP310 as needed.
func (cp *codepage) Decode(b []byte) string {
	runes := make([]rune, 0, len(b))
	var escape bool
	for i := range b {
		if escape {
			escape = false
			if cp.ge2u[b[i]] != '�' {
				runes = append(runes, cp.ge2u[b[i]])
			} else {
				runes = append(runes, 0x1A) // Unicode substitute
			}
		} else {
			// Enter graphic escape mode if necessary.
			if b[i] == cp.ge {
				escape = true
				continue
			}
			// Otherwise perform the mapping.
			runes = append(runes, cp.e2u[b[i]])
		}

	}
	return string(runes) // conversion to UTF-8 is automatic
}

// Encode will convert a UTF-8 Go string into an EBCDIC byte array, handling
// graphic escape to CP310 as needed.
func (cp *codepage) Encode(s string) []byte {
	out := make([]byte, 0, len(s))

	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError {
			break
		}

		if int(r) < len(cp.u2e) {
			// "Fast path" is array look up of Unicode codepoints 0x00-0xFF
			out = append(out, cp.u2e[r])
		} else if v, ok := cp.highu2e[r]; ok {
			// Certain >0xFF code points may be supported in this codepage
			out = append(out, v)
		} else if v, ok := cp.u2ge[r]; ok {
			// include graphic escape character to switch to CP310
			out = append(out, cp.ge, v)
		} else {
			// replacement/substitute character
			out = append(out, cp.esub)
		}
		s = s[size:]
	}

	return out
}

func (cp *codepage) ID() string {
	return cp.id
}

// Certain characters are supported in the "graphic escape" CP310. These are
// arbitrary Unicode code points, so we will look them up via a map. For
// simplicity of our mapping implementation, we will not support the italic
// underlined A-Z characters that require combining characters.
//
// We will share this map among all of the codepages that we provide
// implementations for.
//
// https://public.dhe.ibm.com/software/globalization/gcoc/attachments/CP00310.pdf
var unicodeToCP310 = map[rune]byte{
	'◊': 0x70, '⋄': 0x70, '◆': 0x70, '∧': 0x71, '⋀': 0x71, '¨': 0x72,
	'⌻': 0x73, '⍸': 0x74, '⍷': 0x75, '⊢': 0x76, '⊣': 0x77, '∨': 0x78,
	'∼': 0x80, '║': 0x81, '═': 0x82, '⎸': 0x83, '⎹': 0x84, '│': 0x85,
	'⎥': 0x85, '↑': 0x8A, '↓': 0x8B, '≤': 0x8C, '⌈': 0x8D, '⌊': 0x8E,
	'→': 0x8F, '⎕': 0x90, '▌': 0x91, '▐': 0x92, '▀': 0x93, '▄': 0x94,
	'█': 0x95, '⊃': 0x9A, '⊂': 0x9B, '⌑': 0x9C, '¤': 0x9C, '○': 0x9D,
	'±': 0x9E, '←': 0x9F, '¯': 0xA0, '‾': 0xA0, '°': 0xA1, '─': 0xA2,
	'∙': 0xA3, '•': 0xA3, 'ₙ': 0xA4, '∩': 0xAA, '⋂': 0xAA, '∪': 0xAB,
	'⋃': 0xAB, '⊥': 0xAC, '≥': 0xAE, '∘': 0xAF, '⍺': 0xB0, 'α': 0xB0,
	'∊': 0xB1, '∈': 0xB1, 'ε': 0xB1, '⍳': 0xB2, 'ι': 0xB2, '⍴': 0xB3,
	'ρ': 0xB3, '⍵': 0xB4, 'ω': 0xB4, '×': 0xB6, '∖': 0xB7, '÷': 0xB8,
	'∇': 0xBA, '∆': 0xBB, '⊤': 0xBC, '≠': 0xBE, '∣': 0xBF, '⁽': 0xC1,
	'⁺': 0xC2, '■': 0xC3, '∎': 0xC3, '└': 0xC4, '┌': 0xC5, '├': 0xC6,
	'┴': 0xC7, '⍲': 0xCA, '⍱': 0xCB, '⌷': 0xCC, '⌽': 0xCD, '⍂': 0xCE,
	'⍉': 0xCF, '⁾': 0xD1, '⁻': 0xD2, '┼': 0xD3, '┘': 0xD4, '┐': 0xD5,
	'┤': 0xD6, '┬': 0xD7, '¶': 0xD8, '⌶': 0xDA, 'ǃ': 0xDB, '⍒': 0xDC,
	'⍋': 0xDD, '⍞': 0xDE, '⍝': 0xDF, '≡': 0xE0, '₁': 0xE1, '₂': 0xE2,
	'₃': 0xE3, '⍤': 0xE4, '⍥': 0xE5, '⍪': 0xE6, '€': 0xE7, '⌿': 0xEA,
	'⍀': 0xEB, '∵': 0xEC, '⊖': 0xED, '⌹': 0xEE, '⍕': 0xEF, '⁰': 0xF0,
	'¹': 0xF1, '²': 0xF2, '³': 0xF3, '⁴': 0xF4, '⁵': 0xF5, '⁶': 0xF6,
	'⁷': 0xF7, '⁸': 0xF8, '⁹': 0xF9, '⍫': 0xFB, '⍙': 0xFC, '⍟': 0xFD,
	'⍎': 0xFE,
}

// '�', the Unicode replacement character, is used as a placeholder in byte
// positions that are not assigned in this codepage.
var cp310ToUnicode = []rune{
	/*       x0   x1   x2   x3   x4   x5   x6   x7   x8   x9   xA   xB   xC   xD   xE   xF */
	/* 0x */ '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�',
	/* 1x */ '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�',
	/* 2x */ '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�',
	/* 3x */ '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�',
	/* 4x */ '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�',
	/* 5x */ '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�',
	/* 6x */ '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�', '�',
	/* 7x */ '◊', '∧', '¨', '⌻', '⍸', '⍷', '⊢', '⊣', '∨', '�', '�', '�', '�', '�', '�', '�',
	/* 8x */ '∼', '║', '═', '⎸', '⎹', '⎥', '�', '�', '�', '�', '↑', '↓', '≤', '⌈', '⌊', '→',
	/* 9x */ '⎕', '▌', '▐', '▀', '▄', '█', '�', '�', '�', '�', '⊃', '⊂', '⌑', '○', '±', '←',
	/* Ax */ '‾', '°', '─', '•', 'ₙ', '�', '�', '�', '�', '�', '∩', '⋃', '⊥', '�', '≥', '∘',
	/* Bx */ '⍺', '∈', '⍳', '⍴', 'ω', '�', '×', '∖', '÷', '�', '∇', '∆', '⊤', '�', '≠', '∣',
	/* Cx */ '�', '⁽', '⁺', '■', '└', '┌', '├', '┴', '�', '�', '⍲', '⍱', '⌷', '⌽', '⍂', '⍉',
	/* Dx */ '�', '⁾', '⁻', '┼', '┘', '┐', '┤', '┬', '¶', '�', '⌶', 'ǃ', '⍒', '⍋', '⍞', '⍝',
	/* Ex */ '≡', '₁', '₂', '₃', '⍤', '⍥', '⍪', '€', '�', '�', '⌿', '⍀', '∵', '⊖', '⌹', '⍕',
	/* Fx */ '⁰', '¹', '²', '³', '⁴', '⁵', '⁶', '⁷', '⁸', '⁹', '�', '⍫', '⍙', '⍟', '⍎', '�',
}
