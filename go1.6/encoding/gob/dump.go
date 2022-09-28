// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

package main

// Need to compile package gob with debug.go to build this program.
// See comments in debug.go for how to do this.

import (
	"fmt"
	"os"

	"rsc.io/xstd/go1.6/encoding/gob"
)

func main() {
	var err error
	file := os.Stdin
	if len(os.Args) > 1 {
		file, err = os.Open(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "dump: %s\n", err)
			os.Exit(1)
		}
	}
	gob.Debug(file)
}