// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytes_test

import (
	"os"

	. "rsc.io/xstd/go1.0.3/bytes"
	"rsc.io/xstd/go1.0.3/encoding/base64"
	"rsc.io/xstd/go1.0.3/io"
)

func ExampleBuffer() {
	var b Buffer // A Buffer needs no initialization.
	b.Write([]byte("Hello "))
	b.Write([]byte("world!"))
	b.WriteTo(os.Stdout)
	// Output: Hello world!
}

func ExampleBuffer_reader() {
	// A Buffer can turn a string or a []byte into an io.Reader.
	buf := NewBufferString("R29waGVycyBydWxlIQ==")
	dec := base64.NewDecoder(base64.StdEncoding, buf)
	io.Copy(os.Stdout, dec)
	// Output: Gophers rule!
}
