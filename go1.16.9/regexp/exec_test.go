// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package regexp

import (
	"bufio"
	"compress/bzip2"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"rsc.io/xstd/go1.16.9/internal/testenv"
	"rsc.io/xstd/go1.16.9/io"
	"rsc.io/xstd/go1.16.9/regexp/syntax"
	"rsc.io/xstd/go1.16.9/strconv"
	"rsc.io/xstd/go1.16.9/strings"
	"rsc.io/xstd/go1.16.9/unicode/utf8"
)

// TestRE2 tests this package's regexp API against test cases
// considered during RE2's exhaustive tests, which run all possible
// regexps over a given set of atoms and operators, up to a given
// complexity, over all possible strings over a given alphabet,
// up to a given size. Rather than try to link with RE2, we read a
// log file containing the test cases and the expected matches.
// The log file, re2-exhaustive.txt, is generated by running 'make log'
// in the open source RE2 distribution https://github.com/google/re2/.
//
// The test file format is a sequence of stanzas like:
//
//	strings
//	"abc"
//	"123x"
//	regexps
//	"[a-z]+"
//	0-3;0-3
//	-;-
//	"([0-9])([0-9])([0-9])"
//	-;-
//	-;0-3 0-1 1-2 2-3
//
// The stanza begins by defining a set of strings, quoted
// using Go double-quote syntax, one per line. Then the
// regexps section gives a sequence of regexps to run on
// the strings. In the block that follows a regexp, each line
// gives the semicolon-separated match results of running
// the regexp on the corresponding string.
// Each match result is either a single -, meaning no match, or a
// space-separated sequence of pairs giving the match and
// submatch indices. An unmatched subexpression formats
// its pair as a single - (not illustrated above).  For now
// each regexp run produces two match results, one for a
// ``full match'' that restricts the regexp to matching the entire
// string or nothing, and one for a ``partial match'' that gives
// the leftmost first match found in the string.
//
// Lines beginning with # are comments. Lines beginning with
// a capital letter are test names printed during RE2's test suite
// and are echoed into t but otherwise ignored.
//
// At time of writing, re2-exhaustive.txt is 59 MB but compresses to 385 kB,
// so we store re2-exhaustive.txt.bz2 in the repository and decompress it on the fly.
//
func TestRE2Search(t *testing.T) {
	testRE2(t, "testdata/re2-search.txt")
}

func testRE2(t *testing.T, file string) {
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var txt io.Reader
	if strings.HasSuffix(file, ".bz2") {
		z := bzip2.NewReader(f)
		txt = z
		file = file[:len(file)-len(".bz2")] // for error messages
	} else {
		txt = f
	}
	lineno := 0
	scanner := bufio.NewScanner(txt)
	var (
		str       []string
		input     []string
		inStrings bool
		re        *Regexp
		refull    *Regexp
		nfail     int
		ncase     int
	)
	for lineno := 1; scanner.Scan(); lineno++ {
		line := scanner.Text()
		switch {
		case line == "":
			t.Fatalf("%s:%d: unexpected blank line", file, lineno)
		case line[0] == '#':
			continue
		case 'A' <= line[0] && line[0] <= 'Z':
			// Test name.
			t.Logf("%s\n", line)
			continue
		case line == "strings":
			str = str[:0]
			inStrings = true
		case line == "regexps":
			inStrings = false
		case line[0] == '"':
			q, err := strconv.Unquote(line)
			if err != nil {
				// Fatal because we'll get out of sync.
				t.Fatalf("%s:%d: unquote %s: %v", file, lineno, line, err)
			}
			if inStrings {
				str = append(str, q)
				continue
			}
			// Is a regexp.
			if len(input) != 0 {
				t.Fatalf("%s:%d: out of sync: have %d strings left before %#q", file, lineno, len(input), q)
			}
			re, err = tryCompile(q)
			if err != nil {
				if err.Error() == "error parsing regexp: invalid escape sequence: `\\C`" {
					// We don't and likely never will support \C; keep going.
					continue
				}
				t.Errorf("%s:%d: compile %#q: %v", file, lineno, q, err)
				if nfail++; nfail >= 100 {
					t.Fatalf("stopping after %d errors", nfail)
				}
				continue
			}
			full := `\A(?:` + q + `)\z`
			refull, err = tryCompile(full)
			if err != nil {
				// Fatal because q worked, so this should always work.
				t.Fatalf("%s:%d: compile full %#q: %v", file, lineno, full, err)
			}
			input = str
		case line[0] == '-' || '0' <= line[0] && line[0] <= '9':
			// A sequence of match results.
			ncase++
			if re == nil {
				// Failed to compile: skip results.
				continue
			}
			if len(input) == 0 {
				t.Fatalf("%s:%d: out of sync: no input remaining", file, lineno)
			}
			var text string
			text, input = input[0], input[1:]
			if !isSingleBytes(text) && strings.Contains(re.String(), `\B`) {
				// RE2's \B considers every byte position,
				// so it sees 'not word boundary' in the
				// middle of UTF-8 sequences. This package
				// only considers the positions between runes,
				// so it disagrees. Skip those cases.
				continue
			}
			res := strings.Split(line, ";")
			if len(res) != len(run) {
				t.Fatalf("%s:%d: have %d test results, want %d", file, lineno, len(res), len(run))
			}
			for i := range res {
				have, suffix := run[i](re, refull, text)
				want := parseResult(t, file, lineno, res[i])
				if !same(have, want) {
					t.Errorf("%s:%d: %#q%s.FindSubmatchIndex(%#q) = %v, want %v", file, lineno, re, suffix, text, have, want)
					if nfail++; nfail >= 100 {
						t.Fatalf("stopping after %d errors", nfail)
					}
					continue
				}
				b, suffix := match[i](re, refull, text)
				if b != (want != nil) {
					t.Errorf("%s:%d: %#q%s.MatchString(%#q) = %v, want %v", file, lineno, re, suffix, text, b, !b)
					if nfail++; nfail >= 100 {
						t.Fatalf("stopping after %d errors", nfail)
					}
					continue
				}
			}

		default:
			t.Fatalf("%s:%d: out of sync: %s\n", file, lineno, line)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("%s:%d: %v", file, lineno, err)
	}
	if len(input) != 0 {
		t.Fatalf("%s:%d: out of sync: have %d strings left at EOF", file, lineno, len(input))
	}
	t.Logf("%d cases tested", ncase)
}

var run = []func(*Regexp, *Regexp, string) ([]int, string){
	runFull,
	runPartial,
	runFullLongest,
	runPartialLongest,
}

func runFull(re, refull *Regexp, text string) ([]int, string) {
	refull.longest = false
	return refull.FindStringSubmatchIndex(text), "[full]"
}

func runPartial(re, refull *Regexp, text string) ([]int, string) {
	re.longest = false
	return re.FindStringSubmatchIndex(text), ""
}

func runFullLongest(re, refull *Regexp, text string) ([]int, string) {
	refull.longest = true
	return refull.FindStringSubmatchIndex(text), "[full,longest]"
}

func runPartialLongest(re, refull *Regexp, text string) ([]int, string) {
	re.longest = true
	return re.FindStringSubmatchIndex(text), "[longest]"
}

var match = []func(*Regexp, *Regexp, string) (bool, string){
	matchFull,
	matchPartial,
	matchFullLongest,
	matchPartialLongest,
}

func matchFull(re, refull *Regexp, text string) (bool, string) {
	refull.longest = false
	return refull.MatchString(text), "[full]"
}

func matchPartial(re, refull *Regexp, text string) (bool, string) {
	re.longest = false
	return re.MatchString(text), ""
}

func matchFullLongest(re, refull *Regexp, text string) (bool, string) {
	refull.longest = true
	return refull.MatchString(text), "[full,longest]"
}

func matchPartialLongest(re, refull *Regexp, text string) (bool, string) {
	re.longest = true
	return re.MatchString(text), "[longest]"
}

func isSingleBytes(s string) bool {
	for _, c := range s {
		if c >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func tryCompile(s string) (re *Regexp, err error) {
	// Protect against panic during Compile.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return Compile(s)
}

func parseResult(t *testing.T, file string, lineno int, res string) []int {
	// A single - indicates no match.
	if res == "-" {
		return nil
	}
	// Otherwise, a space-separated list of pairs.
	n := 1
	for j := 0; j < len(res); j++ {
		if res[j] == ' ' {
			n++
		}
	}
	out := make([]int, 2*n)
	i := 0
	n = 0
	for j := 0; j <= len(res); j++ {
		if j == len(res) || res[j] == ' ' {
			// Process a single pair.  - means no submatch.
			pair := res[i:j]
			if pair == "-" {
				out[n] = -1
				out[n+1] = -1
			} else {
				k := strings.Index(pair, "-")
				if k < 0 {
					t.Fatalf("%s:%d: invalid pair %s", file, lineno, pair)
				}
				lo, err1 := strconv.Atoi(pair[:k])
				hi, err2 := strconv.Atoi(pair[k+1:])
				if err1 != nil || err2 != nil || lo > hi {
					t.Fatalf("%s:%d: invalid pair %s", file, lineno, pair)
				}
				out[n] = lo
				out[n+1] = hi
			}
			n += 2
			i = j + 1
		}
	}
	return out
}

func same(x, y []int) bool {
	if len(x) != len(y) {
		return false
	}
	for i, xi := range x {
		if xi != y[i] {
			return false
		}
	}
	return true
}

// TestFowler runs this package's regexp API against the
// POSIX regular expression tests collected by Glenn Fowler
// at http://www2.research.att.com/~astopen/testregex/testregex.html.
func TestFowler(t *testing.T) {
	files, err := filepath.Glob("testdata/*.dat")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		t.Log(file)
		testFowler(t, file)
	}
}

var notab = MustCompilePOSIX(`[^\t]+`)

func testFowler(t *testing.T, file string) {
	f, err := os.Open(file)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	b := bufio.NewReader(f)
	lineno := 0
	lastRegexp := ""
Reading:
	for {
		lineno++
		line, err := b.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				t.Errorf("%s:%d: %v", file, lineno, err)
			}
			break Reading
		}

		// http://www2.research.att.com/~astopen/man/man1/testregex.html
		//
		// INPUT FORMAT
		//   Input lines may be blank, a comment beginning with #, or a test
		//   specification. A specification is five fields separated by one
		//   or more tabs. NULL denotes the empty string and NIL denotes the
		//   0 pointer.
		if line[0] == '#' || line[0] == '\n' {
			continue Reading
		}
		line = line[:len(line)-1]
		field := notab.FindAllString(line, -1)
		for i, f := range field {
			if f == "NULL" {
				field[i] = ""
			}
			if f == "NIL" {
				t.Logf("%s:%d: skip: %s", file, lineno, line)
				continue Reading
			}
		}
		if len(field) == 0 {
			continue Reading
		}

		//   Field 1: the regex(3) flags to apply, one character per REG_feature
		//   flag. The test is skipped if REG_feature is not supported by the
		//   implementation. If the first character is not [BEASKLP] then the
		//   specification is a global control line. One or more of [BEASKLP] may be
		//   specified; the test will be repeated for each mode.
		//
		//     B 	basic			BRE	(grep, ed, sed)
		//     E 	REG_EXTENDED		ERE	(egrep)
		//     A	REG_AUGMENTED		ARE	(egrep with negation)
		//     S	REG_SHELL		SRE	(sh glob)
		//     K	REG_SHELL|REG_AUGMENTED	KRE	(ksh glob)
		//     L	REG_LITERAL		LRE	(fgrep)
		//
		//     a	REG_LEFT|REG_RIGHT	implicit ^...$
		//     b	REG_NOTBOL		lhs does not match ^
		//     c	REG_COMMENT		ignore space and #...\n
		//     d	REG_SHELL_DOT		explicit leading . match
		//     e	REG_NOTEOL		rhs does not match $
		//     f	REG_MULTIPLE		multiple \n separated patterns
		//     g	FNM_LEADING_DIR		testfnmatch only -- match until /
		//     h	REG_MULTIREF		multiple digit backref
		//     i	REG_ICASE		ignore case
		//     j	REG_SPAN		. matches \n
		//     k	REG_ESCAPE		\ to escape [...] delimiter
		//     l	REG_LEFT		implicit ^...
		//     m	REG_MINIMAL		minimal match
		//     n	REG_NEWLINE		explicit \n match
		//     o	REG_ENCLOSED		(|&) magic inside [@|&](...)
		//     p	REG_SHELL_PATH		explicit / match
		//     q	REG_DELIMITED		delimited pattern
		//     r	REG_RIGHT		implicit ...$
		//     s	REG_SHELL_ESCAPED	\ not special
		//     t	REG_MUSTDELIM		all delimiters must be specified
		//     u	standard unspecified behavior -- errors not counted
		//     v	REG_CLASS_ESCAPE	\ special inside [...]
		//     w	REG_NOSUB		no subexpression match array
		//     x	REG_LENIENT		let some errors slide
		//     y	REG_LEFT		regexec() implicit ^...
		//     z	REG_NULL		NULL subexpressions ok
		//     $	                        expand C \c escapes in fields 2 and 3
		//     /	                        field 2 is a regsubcomp() expression
		//     =	                        field 3 is a regdecomp() expression
		//
		//   Field 1 control lines:
		//
		//     C		set LC_COLLATE and LC_CTYPE to locale in field 2
		//
		//     ?test ...	output field 5 if passed and != EXPECTED, silent otherwise
		//     &test ...	output field 5 if current and previous passed
		//     |test ...	output field 5 if current passed and previous failed
		//     ; ...	output field 2 if previous failed
		//     {test ...	skip if failed until }
		//     }		end of skip
		//
		//     : comment		comment copied as output NOTE
		//     :comment:test	:comment: ignored
		//     N[OTE] comment	comment copied as output NOTE
		//     T[EST] comment	comment
		//
		//     number		use number for nmatch (20 by default)
		flag := field[0]
		switch flag[0] {
		case '?', '&', '|', ';', '{', '}':
			// Ignore all the control operators.
			// Just run everything.
			flag = flag[1:]
			if flag == "" {
				continue Reading
			}
		case ':':
			i := strings.Index(flag[1:], ":")
			if i < 0 {
				t.Logf("skip: %s", line)
				continue Reading
			}
			flag = flag[1+i+1:]
		case 'C', 'N', 'T', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			t.Logf("skip: %s", line)
			continue Reading
		}

		// Can check field count now that we've handled the myriad comment formats.
		if len(field) < 4 {
			t.Errorf("%s:%d: too few fields: %s", file, lineno, line)
			continue Reading
		}

		// Expand C escapes (a.k.a. Go escapes).
		if strings.Contains(flag, "$") {
			f := `"` + field[1] + `"`
			if field[1], err = strconv.Unquote(f); err != nil {
				t.Errorf("%s:%d: cannot unquote %s", file, lineno, f)
			}
			f = `"` + field[2] + `"`
			if field[2], err = strconv.Unquote(f); err != nil {
				t.Errorf("%s:%d: cannot unquote %s", file, lineno, f)
			}
		}

		//   Field 2: the regular expression pattern; SAME uses the pattern from
		//     the previous specification.
		//
		if field[1] == "SAME" {
			field[1] = lastRegexp
		}
		lastRegexp = field[1]

		//   Field 3: the string to match.
		text := field[2]

		//   Field 4: the test outcome...
		ok, shouldCompile, shouldMatch, pos := parseFowlerResult(field[3])
		if !ok {
			t.Errorf("%s:%d: cannot parse result %#q", file, lineno, field[3])
			continue Reading
		}

		//   Field 5: optional comment appended to the report.

	Testing:
		// Run test once for each specified capital letter mode that we support.
		for _, c := range flag {
			pattern := field[1]
			syn := syntax.POSIX | syntax.ClassNL
			switch c {
			default:
				continue Testing
			case 'E':
				// extended regexp (what we support)
			case 'L':
				// literal
				pattern = QuoteMeta(pattern)
			}

			for _, c := range flag {
				switch c {
				case 'i':
					syn |= syntax.FoldCase
				}
			}

			re, err := compile(pattern, syn, true)
			if err != nil {
				if shouldCompile {
					t.Errorf("%s:%d: %#q did not compile", file, lineno, pattern)
				}
				continue Testing
			}
			if !shouldCompile {
				t.Errorf("%s:%d: %#q should not compile", file, lineno, pattern)
				continue Testing
			}
			match := re.MatchString(text)
			if match != shouldMatch {
				t.Errorf("%s:%d: %#q.Match(%#q) = %v, want %v", file, lineno, pattern, text, match, shouldMatch)
				continue Testing
			}
			have := re.FindStringSubmatchIndex(text)
			if (len(have) > 0) != match {
				t.Errorf("%s:%d: %#q.Match(%#q) = %v, but %#q.FindSubmatchIndex(%#q) = %v", file, lineno, pattern, text, match, pattern, text, have)
				continue Testing
			}
			if len(have) > len(pos) {
				have = have[:len(pos)]
			}
			if !same(have, pos) {
				t.Errorf("%s:%d: %#q.FindSubmatchIndex(%#q) = %v, want %v", file, lineno, pattern, text, have, pos)
			}
		}
	}
}

func parseFowlerResult(s string) (ok, compiled, matched bool, pos []int) {
	//   Field 4: the test outcome. This is either one of the posix error
	//     codes (with REG_ omitted) or the match array, a list of (m,n)
	//     entries with m and n being first and last+1 positions in the
	//     field 3 string, or NULL if REG_NOSUB is in effect and success
	//     is expected. BADPAT is acceptable in place of any regcomp(3)
	//     error code. The match[] array is initialized to (-2,-2) before
	//     each test. All array elements from 0 to nmatch-1 must be specified
	//     in the outcome. Unspecified endpoints (offset -1) are denoted by ?.
	//     Unset endpoints (offset -2) are denoted by X. {x}(o:n) denotes a
	//     matched (?{...}) expression, where x is the text enclosed by {...},
	//     o is the expression ordinal counting from 1, and n is the length of
	//     the unmatched portion of the subject string. If x starts with a
	//     number then that is the return value of re_execf(), otherwise 0 is
	//     returned.
	switch {
	case s == "":
		// Match with no position information.
		ok = true
		compiled = true
		matched = true
		return
	case s == "NOMATCH":
		// Match failure.
		ok = true
		compiled = true
		matched = false
		return
	case 'A' <= s[0] && s[0] <= 'Z':
		// All the other error codes are compile errors.
		ok = true
		compiled = false
		return
	}
	compiled = true

	var x []int
	for s != "" {
		var end byte = ')'
		if len(x)%2 == 0 {
			if s[0] != '(' {
				ok = false
				return
			}
			s = s[1:]
			end = ','
		}
		i := 0
		for i < len(s) && s[i] != end {
			i++
		}
		if i == 0 || i == len(s) {
			ok = false
			return
		}
		var v = -1
		var err error
		if s[:i] != "?" {
			v, err = strconv.Atoi(s[:i])
			if err != nil {
				ok = false
				return
			}
		}
		x = append(x, v)
		s = s[i+1:]
	}
	if len(x)%2 != 0 {
		ok = false
		return
	}
	ok = true
	matched = true
	pos = x
	return
}

var text []byte

func makeText(n int) []byte {
	if len(text) >= n {
		return text[:n]
	}
	text = make([]byte, n)
	x := ^uint32(0)
	for i := range text {
		x += x
		x ^= 1
		if int32(x) < 0 {
			x ^= 0x88888eef
		}
		if x%31 == 0 {
			text[i] = '\n'
		} else {
			text[i] = byte(x%(0x7E+1-0x20) + 0x20)
		}
	}
	return text
}

func BenchmarkMatch(b *testing.B) {
	isRaceBuilder := strings.HasSuffix(testenv.Builder(), "-race")

	for _, data := range benchData {
		r := MustCompile(data.re)
		for _, size := range benchSizes {
			if (isRaceBuilder || testing.Short()) && size.n > 1<<10 {
				continue
			}
			t := makeText(size.n)
			b.Run(data.name+"/"+size.name, func(b *testing.B) {
				b.SetBytes(int64(size.n))
				for i := 0; i < b.N; i++ {
					if r.Match(t) {
						b.Fatal("match!")
					}
				}
			})
		}
	}
}

func BenchmarkMatch_onepass_regex(b *testing.B) {
	isRaceBuilder := strings.HasSuffix(testenv.Builder(), "-race")
	r := MustCompile(`(?s)\A.*\z`)
	if r.onepass == nil {
		b.Fatalf("want onepass regex, but %q is not onepass", r)
	}
	for _, size := range benchSizes {
		if (isRaceBuilder || testing.Short()) && size.n > 1<<10 {
			continue
		}
		t := makeText(size.n)
		b.Run(size.name, func(b *testing.B) {
			b.SetBytes(int64(size.n))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if !r.Match(t) {
					b.Fatal("not match!")
				}
			}
		})
	}
}

var benchData = []struct{ name, re string }{
	{"Easy0", "ABCDEFGHIJKLMNOPQRSTUVWXYZ$"},
	{"Easy0i", "(?i)ABCDEFGHIJklmnopqrstuvwxyz$"},
	{"Easy1", "A[AB]B[BC]C[CD]D[DE]E[EF]F[FG]G[GH]H[HI]I[IJ]J$"},
	{"Medium", "[XYZ]ABCDEFGHIJKLMNOPQRSTUVWXYZ$"},
	{"Hard", "[ -~]*ABCDEFGHIJKLMNOPQRSTUVWXYZ$"},
	{"Hard1", "ABCD|CDEF|EFGH|GHIJ|IJKL|KLMN|MNOP|OPQR|QRST|STUV|UVWX|WXYZ"},
}

var benchSizes = []struct {
	name string
	n    int
}{
	{"16", 16},
	{"32", 32},
	{"1K", 1 << 10},
	{"32K", 32 << 10},
	{"1M", 1 << 20},
	{"32M", 32 << 20},
}

func TestLongest(t *testing.T) {
	re, err := Compile(`a(|b)`)
	if err != nil {
		t.Fatal(err)
	}
	if g, w := re.FindString("ab"), "a"; g != w {
		t.Errorf("first match was %q, want %q", g, w)
	}
	re.Longest()
	if g, w := re.FindString("ab"), "ab"; g != w {
		t.Errorf("longest match was %q, want %q", g, w)
	}
}

// TestProgramTooLongForBacktrack tests that a regex which is too long
// for the backtracker still executes properly.
func TestProgramTooLongForBacktrack(t *testing.T) {
	longRegex := MustCompile(`(one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty|twentyone|twentytwo|twentythree|twentyfour|twentyfive|twentysix|twentyseven|twentyeight|twentynine|thirty|thirtyone|thirtytwo|thirtythree|thirtyfour|thirtyfive|thirtysix|thirtyseven|thirtyeight|thirtynine|forty|fortyone|fortytwo|fortythree|fortyfour|fortyfive|fortysix|fortyseven|fortyeight|fortynine|fifty|fiftyone|fiftytwo|fiftythree|fiftyfour|fiftyfive|fiftysix|fiftyseven|fiftyeight|fiftynine|sixty|sixtyone|sixtytwo|sixtythree|sixtyfour|sixtyfive|sixtysix|sixtyseven|sixtyeight|sixtynine|seventy|seventyone|seventytwo|seventythree|seventyfour|seventyfive|seventysix|seventyseven|seventyeight|seventynine|eighty|eightyone|eightytwo|eightythree|eightyfour|eightyfive|eightysix|eightyseven|eightyeight|eightynine|ninety|ninetyone|ninetytwo|ninetythree|ninetyfour|ninetyfive|ninetysix|ninetyseven|ninetyeight|ninetynine|onehundred)`)
	if !longRegex.MatchString("two") {
		t.Errorf("longRegex.MatchString(\"two\") was false, want true")
	}
	if longRegex.MatchString("xxx") {
		t.Errorf("longRegex.MatchString(\"xxx\") was true, want false")
	}
}
