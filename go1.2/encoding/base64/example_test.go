// Copyright 2012 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Keep in sync with ../base32/example_test.go.

package base64_test

import (
	"fmt"
	"os"

	"rsc.io/xstd/go1.2/encoding/base64"
)

func ExampleEncoding_EncodeToString() {
	data := []byte("any + old & data")
	str := base64.StdEncoding.EncodeToString(data)
	fmt.Println(str)
	// Output:
	// YW55ICsgb2xkICYgZGF0YQ==
}

func ExampleEncoding_DecodeString() {
	str := "c29tZSBkYXRhIHdpdGggACBhbmQg77u/"
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%q\n", data)
	// Output:
	// "some data with \x00 and \ufeff"
}

func ExampleNewEncoder() {
	input := []byte("foo\x00bar")
	encoder := base64.NewEncoder(base64.StdEncoding, os.Stdout)
	encoder.Write(input)
	// Must close the encoder when finished to flush any partial blocks.
	// If you comment out the following line, the last partial block "r"
	// won't be encoded.
	encoder.Close()
	// Output:
	// Zm9vAGJhcg==
}
