// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytes

import "bytes"

// IndexByte returns the index of the first instance of c in s, or -1 if c is not present in s.
func IndexByte(s []byte, c byte) int {
	return bytes.IndexByte(s, c)
}

// Equal returns a boolean reporting whether a == b.
// A nil argument is equivalent to an empty slice.
func Equal(a, b []byte) bool {
	return bytes.Equal(a, b)
}
