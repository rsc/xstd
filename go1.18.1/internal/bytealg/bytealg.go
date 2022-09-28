// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytealg

import (
	"bytes"
	"strings"
)

func Compare(a, b []byte) int              { return bytes.Compare(a, b) }
func Count(b, c []byte) int                { return bytes.Count(b, c) }
func CountString(s, c string) int          { return strings.Count(s, c) }
func Equal(a, b []byte) bool               { return bytes.Equal(a, b) }
func Index(a, b []byte) int                { return bytes.Index(a, b) }
func IndexByte(b []byte, c byte) int       { return bytes.IndexByte(b, c) }
func IndexByteString(s string, c byte) int { return strings.IndexByte(s, c) }
func IndexString(a, b string) int          { return strings.Index(a, b) }
