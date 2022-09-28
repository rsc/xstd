// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytealg

import (
	"bytes"
	"strings"
)

func Compare(a, b []byte) int              { return bytes.Compare(a, b) }
func Equal(a, b []byte) bool               { return bytes.Equal(a, b) }
func Index(a, b []byte) int                { return bytes.Index(a, b) }
func IndexByte(b []byte, c byte) int       { return bytes.IndexByte(b, c) }
func IndexByteString(s string, c byte) int { return strings.IndexByte(s, c) }
func IndexString(a, b string) int          { return strings.Index(a, b) }

const (
	MaxBruteForce = 64
	PrimeRK       = 16777619
	MaxLen        = 31
)

func Cutover(n int) int {
	// 1 error per 8 characters, plus a few slop to start.
	return (n + 16) / 8
}

func CountString(s string, c byte) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			n++
		}
	}
	return n
}

func Count(b []byte, c byte) int {
	n := 0
	for _, x := range b {
		if x == c {
			n++
		}
	}
	return n
}
