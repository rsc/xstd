// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cpu_test

import (
	"runtime"
	"testing"

	"rsc.io/xstd/go1.9.4/internal/cpu"
)

func TestAMD64minimalFeatures(t *testing.T) {
	if runtime.GOARCH == "amd64" {
		if !cpu.X86.HasSSE2 {
			t.Fatalf("HasSSE2 expected true, got false")
		}
	}
}

func TestAVX2hasAVX(t *testing.T) {
	if runtime.GOARCH == "amd64" {
		if cpu.X86.HasAVX2 && !cpu.X86.HasAVX {
			t.Fatalf("HasAVX expected true, got false")
		}
	}
}
