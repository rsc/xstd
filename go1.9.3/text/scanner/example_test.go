// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scanner_test

import (
	"fmt"

	"rsc.io/xstd/go1.9.3/strings"
	"rsc.io/xstd/go1.9.3/text/scanner"
)

func Example() {
	const src = `
// This is scanned code.
if a > 10 {
	someParsable = text
}`
	var s scanner.Scanner
	s.Init(strings.NewReader(src))
	s.Filename = "example"
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		fmt.Printf("%s: %s\n", s.Position, s.TokenText())
	}

	// Output:
	// example:3:1: if
	// example:3:4: a
	// example:3:6: >
	// example:3:8: 10
	// example:3:11: {
	// example:4:2: someParsable
	// example:4:15: =
	// example:4:17: text
	// example:5:1: }
}
