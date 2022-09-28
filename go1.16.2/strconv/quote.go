// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run makeisprint.go -output isprint.go

package strconv

import (
	"rsc.io/xstd/go1.16.2/internal/bytealg"
	"rsc.io/xstd/go1.16.2/unicode/utf8"
)

const (
	lowerhex = "0123456789abcdef"
	upperhex = "0123456789ABCDEF"
)

func quoteWith(s string, quote byte, ASCIIonly, graphicOnly bool) string {
	return string(appendQuotedWith(make([]byte, 0, 3*len(s)/2), s, quote, ASCIIonly, graphicOnly))
}

func quoteRuneWith(r rune, quote byte, ASCIIonly, graphicOnly bool) string {
	return string(appendQuotedRuneWith(nil, r, quote, ASCIIonly, graphicOnly))
}

func appendQuotedWith(buf []byte, s string, quote byte, ASCIIonly, graphicOnly bool) []byte {
	// Often called with big strings, so preallocate. If there's quoting,
	// this is conservative but still helps a lot.
	if cap(buf)-len(buf) < len(s) {
		nBuf := make([]byte, len(buf), len(buf)+1+len(s)+1)
		copy(nBuf, buf)
		buf = nBuf
	}
	buf = append(buf, quote)
	for width := 0; len(s) > 0; s = s[width:] {
		r := rune(s[0])
		width = 1
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}
		if width == 1 && r == utf8.RuneError {
			buf = append(buf, `\x`...)
			buf = append(buf, lowerhex[s[0]>>4])
			buf = append(buf, lowerhex[s[0]&0xF])
			continue
		}
		buf = appendEscapedRune(buf, r, quote, ASCIIonly, graphicOnly)
	}
	buf = append(buf, quote)
	return buf
}

func appendQuotedRuneWith(buf []byte, r rune, quote byte, ASCIIonly, graphicOnly bool) []byte {
	buf = append(buf, quote)
	if !utf8.ValidRune(r) {
		r = utf8.RuneError
	}
	buf = appendEscapedRune(buf, r, quote, ASCIIonly, graphicOnly)
	buf = append(buf, quote)
	return buf
}

func appendEscapedRune(buf []byte, r rune, quote byte, ASCIIonly, graphicOnly bool) []byte {
	var runeTmp [utf8.UTFMax]byte
	if r == rune(quote) || r == '\\' { // always backslashed
		buf = append(buf, '\\')
		buf = append(buf, byte(r))
		return buf
	}
	if ASCIIonly {
		if r < utf8.RuneSelf && IsPrint(r) {
			buf = append(buf, byte(r))
			return buf
		}
	} else if IsPrint(r) || graphicOnly && isInGraphicList(r) {
		n := utf8.EncodeRune(runeTmp[:], r)
		buf = append(buf, runeTmp[:n]...)
		return buf
	}
	switch r {
	case '\a':
		buf = append(buf, `\a`...)
	case '\b':
		buf = append(buf, `\b`...)
	case '\f':
		buf = append(buf, `\f`...)
	case '\n':
		buf = append(buf, `\n`...)
	case '\r':
		buf = append(buf, `\r`...)
	case '\t':
		buf = append(buf, `\t`...)
	case '\v':
		buf = append(buf, `\v`...)
	default:
		switch {
		case r < ' ':
			buf = append(buf, `\x`...)
			buf = append(buf, lowerhex[byte(r)>>4])
			buf = append(buf, lowerhex[byte(r)&0xF])
		case r > utf8.MaxRune:
			r = 0xFFFD
			fallthrough
		case r < 0x10000:
			buf = append(buf, `\u`...)
			for s := 12; s >= 0; s -= 4 {
				buf = append(buf, lowerhex[r>>uint(s)&0xF])
			}
		default:
			buf = append(buf, `\U`...)
			for s := 28; s >= 0; s -= 4 {
				buf = append(buf, lowerhex[r>>uint(s)&0xF])
			}
		}
	}
	return buf
}

// Quote returns a double-quoted Go string literal representing s. The
// returned string uses Go escape sequences (\t, \n, \xFF, \u0100) for
// control characters and non-printable characters as defined by
// IsPrint.
func Quote(s string) string {
	return quoteWith(s, '"', false, false)
}

// AppendQuote appends a double-quoted Go string literal representing s,
// as generated by Quote, to dst and returns the extended buffer.
func AppendQuote(dst []byte, s string) []byte {
	return appendQuotedWith(dst, s, '"', false, false)
}

// QuoteToASCII returns a double-quoted Go string literal representing s.
// The returned string uses Go escape sequences (\t, \n, \xFF, \u0100) for
// non-ASCII characters and non-printable characters as defined by IsPrint.
func QuoteToASCII(s string) string {
	return quoteWith(s, '"', true, false)
}

// AppendQuoteToASCII appends a double-quoted Go string literal representing s,
// as generated by QuoteToASCII, to dst and returns the extended buffer.
func AppendQuoteToASCII(dst []byte, s string) []byte {
	return appendQuotedWith(dst, s, '"', true, false)
}

// QuoteToGraphic returns a double-quoted Go string literal representing s.
// The returned string leaves Unicode graphic characters, as defined by
// IsGraphic, unchanged and uses Go escape sequences (\t, \n, \xFF, \u0100)
// for non-graphic characters.
func QuoteToGraphic(s string) string {
	return quoteWith(s, '"', false, true)
}

// AppendQuoteToGraphic appends a double-quoted Go string literal representing s,
// as generated by QuoteToGraphic, to dst and returns the extended buffer.
func AppendQuoteToGraphic(dst []byte, s string) []byte {
	return appendQuotedWith(dst, s, '"', false, true)
}

// QuoteRune returns a single-quoted Go character literal representing the
// rune. The returned string uses Go escape sequences (\t, \n, \xFF, \u0100)
// for control characters and non-printable characters as defined by IsPrint.
func QuoteRune(r rune) string {
	return quoteRuneWith(r, '\'', false, false)
}

// AppendQuoteRune appends a single-quoted Go character literal representing the rune,
// as generated by QuoteRune, to dst and returns the extended buffer.
func AppendQuoteRune(dst []byte, r rune) []byte {
	return appendQuotedRuneWith(dst, r, '\'', false, false)
}

// QuoteRuneToASCII returns a single-quoted Go character literal representing
// the rune. The returned string uses Go escape sequences (\t, \n, \xFF,
// \u0100) for non-ASCII characters and non-printable characters as defined
// by IsPrint.
func QuoteRuneToASCII(r rune) string {
	return quoteRuneWith(r, '\'', true, false)
}

// AppendQuoteRuneToASCII appends a single-quoted Go character literal representing the rune,
// as generated by QuoteRuneToASCII, to dst and returns the extended buffer.
func AppendQuoteRuneToASCII(dst []byte, r rune) []byte {
	return appendQuotedRuneWith(dst, r, '\'', true, false)
}

// QuoteRuneToGraphic returns a single-quoted Go character literal representing
// the rune. If the rune is not a Unicode graphic character,
// as defined by IsGraphic, the returned string will use a Go escape sequence
// (\t, \n, \xFF, \u0100).
func QuoteRuneToGraphic(r rune) string {
	return quoteRuneWith(r, '\'', false, true)
}

// AppendQuoteRuneToGraphic appends a single-quoted Go character literal representing the rune,
// as generated by QuoteRuneToGraphic, to dst and returns the extended buffer.
func AppendQuoteRuneToGraphic(dst []byte, r rune) []byte {
	return appendQuotedRuneWith(dst, r, '\'', false, true)
}

// CanBackquote reports whether the string s can be represented
// unchanged as a single-line backquoted string without control
// characters other than tab.
func CanBackquote(s string) bool {
	for len(s) > 0 {
		r, wid := utf8.DecodeRuneInString(s)
		s = s[wid:]
		if wid > 1 {
			if r == '\ufeff' {
				return false // BOMs are invisible and should not be quoted.
			}
			continue // All other multibyte runes are correctly encoded and assumed printable.
		}
		if r == utf8.RuneError {
			return false
		}
		if (r < ' ' && r != '\t') || r == '`' || r == '\u007F' {
			return false
		}
	}
	return true
}

func unhex(b byte) (v rune, ok bool) {
	c := rune(b)
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return
}

// UnquoteChar decodes the first character or byte in the escaped string
// or character literal represented by the string s.
// It returns four values:
//
//	1) value, the decoded Unicode code point or byte value;
//	2) multibyte, a boolean indicating whether the decoded character requires a multibyte UTF-8 representation;
//	3) tail, the remainder of the string after the character; and
//	4) an error that will be nil if the character is syntactically valid.
//
// The second argument, quote, specifies the type of literal being parsed
// and therefore which escaped quote character is permitted.
// If set to a single quote, it permits the sequence \' and disallows unescaped '.
// If set to a double quote, it permits \" and disallows unescaped ".
// If set to zero, it does not permit either escape and allows both quote characters to appear unescaped.
func UnquoteChar(s string, quote byte) (value rune, multibyte bool, tail string, err error) {
	// easy cases
	if len(s) == 0 {
		err = ErrSyntax
		return
	}
	switch c := s[0]; {
	case c == quote && (quote == '\'' || quote == '"'):
		err = ErrSyntax
		return
	case c >= utf8.RuneSelf:
		r, size := utf8.DecodeRuneInString(s)
		return r, true, s[size:], nil
	case c != '\\':
		return rune(s[0]), false, s[1:], nil
	}

	// hard case: c is backslash
	if len(s) <= 1 {
		err = ErrSyntax
		return
	}
	c := s[1]
	s = s[2:]

	switch c {
	case 'a':
		value = '\a'
	case 'b':
		value = '\b'
	case 'f':
		value = '\f'
	case 'n':
		value = '\n'
	case 'r':
		value = '\r'
	case 't':
		value = '\t'
	case 'v':
		value = '\v'
	case 'x', 'u', 'U':
		n := 0
		switch c {
		case 'x':
			n = 2
		case 'u':
			n = 4
		case 'U':
			n = 8
		}
		var v rune
		if len(s) < n {
			err = ErrSyntax
			return
		}
		for j := 0; j < n; j++ {
			x, ok := unhex(s[j])
			if !ok {
				err = ErrSyntax
				return
			}
			v = v<<4 | x
		}
		s = s[n:]
		if c == 'x' {
			// single-byte string, possibly not UTF-8
			value = v
			break
		}
		if v > utf8.MaxRune {
			err = ErrSyntax
			return
		}
		value = v
		multibyte = true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		v := rune(c) - '0'
		if len(s) < 2 {
			err = ErrSyntax
			return
		}
		for j := 0; j < 2; j++ { // one digit already; two more
			x := rune(s[j]) - '0'
			if x < 0 || x > 7 {
				err = ErrSyntax
				return
			}
			v = (v << 3) | x
		}
		s = s[2:]
		if v > 255 {
			err = ErrSyntax
			return
		}
		value = v
	case '\\':
		value = '\\'
	case '\'', '"':
		if c != quote {
			err = ErrSyntax
			return
		}
		value = rune(c)
	default:
		err = ErrSyntax
		return
	}
	tail = s
	return
}

// Unquote interprets s as a single-quoted, double-quoted,
// or backquoted Go string literal, returning the string value
// that s quotes.  (If s is single-quoted, it would be a Go
// character literal; Unquote returns the corresponding
// one-character string.)
func Unquote(s string) (string, error) {
	n := len(s)
	if n < 2 {
		return "", ErrSyntax
	}
	quote := s[0]
	if quote != s[n-1] {
		return "", ErrSyntax
	}
	s = s[1 : n-1]

	if quote == '`' {
		if contains(s, '`') {
			return "", ErrSyntax
		}
		if contains(s, '\r') {
			// -1 because we know there is at least one \r to remove.
			buf := make([]byte, 0, len(s)-1)
			for i := 0; i < len(s); i++ {
				if s[i] != '\r' {
					buf = append(buf, s[i])
				}
			}
			return string(buf), nil
		}
		return s, nil
	}
	if quote != '"' && quote != '\'' {
		return "", ErrSyntax
	}
	if contains(s, '\n') {
		return "", ErrSyntax
	}

	// Is it trivial? Avoid allocation.
	if !contains(s, '\\') && !contains(s, quote) {
		switch quote {
		case '"':
			if utf8.ValidString(s) {
				return s, nil
			}
		case '\'':
			r, size := utf8.DecodeRuneInString(s)
			if size == len(s) && (r != utf8.RuneError || size != 1) {
				return s, nil
			}
		}
	}

	var runeTmp [utf8.UTFMax]byte
	buf := make([]byte, 0, 3*len(s)/2) // Try to avoid more allocations.
	for len(s) > 0 {
		c, multibyte, ss, err := UnquoteChar(s, quote)
		if err != nil {
			return "", err
		}
		s = ss
		if c < utf8.RuneSelf || !multibyte {
			buf = append(buf, byte(c))
		} else {
			n := utf8.EncodeRune(runeTmp[:], c)
			buf = append(buf, runeTmp[:n]...)
		}
		if quote == '\'' && len(s) != 0 {
			// single-quoted must be single character
			return "", ErrSyntax
		}
	}
	return string(buf), nil
}

// contains reports whether the string contains the byte c.
func contains(s string, c byte) bool {
	return bytealg.IndexByteString(s, c) != -1
}

// bsearch16 returns the smallest i such that a[i] >= x.
// If there is no such i, bsearch16 returns len(a).
func bsearch16(a []uint16, x uint16) int {
	i, j := 0, len(a)
	for i < j {
		h := i + (j-i)/2
		if a[h] < x {
			i = h + 1
		} else {
			j = h
		}
	}
	return i
}

// bsearch32 returns the smallest i such that a[i] >= x.
// If there is no such i, bsearch32 returns len(a).
func bsearch32(a []uint32, x uint32) int {
	i, j := 0, len(a)
	for i < j {
		h := i + (j-i)/2
		if a[h] < x {
			i = h + 1
		} else {
			j = h
		}
	}
	return i
}

// TODO: IsPrint is a local implementation of unicode.IsPrint, verified by the tests
// to give the same answer. It allows this package not to depend on unicode,
// and therefore not pull in all the Unicode tables. If the linker were better
// at tossing unused tables, we could get rid of this implementation.
// That would be nice.

// IsPrint reports whether the rune is defined as printable by Go, with
// the same definition as unicode.IsPrint: letters, numbers, punctuation,
// symbols and ASCII space.
func IsPrint(r rune) bool {
	// Fast check for Latin-1
	if r <= 0xFF {
		if 0x20 <= r && r <= 0x7E {
			// All the ASCII is printable from space through DEL-1.
			return true
		}
		if 0xA1 <= r && r <= 0xFF {
			// Similarly for ¡ through ÿ...
			return r != 0xAD // ...except for the bizarre soft hyphen.
		}
		return false
	}

	// Same algorithm, either on uint16 or uint32 value.
	// First, find first i such that isPrint[i] >= x.
	// This is the index of either the start or end of a pair that might span x.
	// The start is even (isPrint[i&^1]) and the end is odd (isPrint[i|1]).
	// If we find x in a range, make sure x is not in isNotPrint list.

	if 0 <= r && r < 1<<16 {
		rr, isPrint, isNotPrint := uint16(r), isPrint16, isNotPrint16
		i := bsearch16(isPrint, rr)
		if i >= len(isPrint) || rr < isPrint[i&^1] || isPrint[i|1] < rr {
			return false
		}
		j := bsearch16(isNotPrint, rr)
		return j >= len(isNotPrint) || isNotPrint[j] != rr
	}

	rr, isPrint, isNotPrint := uint32(r), isPrint32, isNotPrint32
	i := bsearch32(isPrint, rr)
	if i >= len(isPrint) || rr < isPrint[i&^1] || isPrint[i|1] < rr {
		return false
	}
	if r >= 0x20000 {
		return true
	}
	r -= 0x10000
	j := bsearch16(isNotPrint, uint16(r))
	return j >= len(isNotPrint) || isNotPrint[j] != uint16(r)
}

// IsGraphic reports whether the rune is defined as a Graphic by Unicode. Such
// characters include letters, marks, numbers, punctuation, symbols, and
// spaces, from categories L, M, N, P, S, and Zs.
func IsGraphic(r rune) bool {
	if IsPrint(r) {
		return true
	}
	return isInGraphicList(r)
}

// isInGraphicList reports whether the rune is in the isGraphic list. This separation
// from IsGraphic allows quoteWith to avoid two calls to IsPrint.
// Should be called only if IsPrint fails.
func isInGraphicList(r rune) bool {
	// We know r must fit in 16 bits - see makeisprint.go.
	if r > 0xFFFF {
		return false
	}
	rr := uint16(r)
	i := bsearch16(isGraphic, rr)
	return i < len(isGraphic) && rr == isGraphic[i]
}
