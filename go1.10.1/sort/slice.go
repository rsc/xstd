// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !compiler_bootstrap || go1.8
// +build !compiler_bootstrap go1.8

package sort

import "reflect"

// Slice sorts the provided slice given the provided less function.
//
// The sort is not guaranteed to be stable. For a stable sort, use
// SliceStable.
//
// The function panics if the provided interface is not a slice.
func Slice(slice interface{}, less func(i, j int) bool) {
	rv := reflect.ValueOf(slice)
	swap := reflect.Swapper(slice)
	length := rv.Len()
	quickSort_func(lessSwap{less, swap}, 0, length, maxDepth(length))
}

// SliceStable sorts the provided slice given the provided less
// function while keeping the original order of equal elements.
//
// The function panics if the provided interface is not a slice.
func SliceStable(slice interface{}, less func(i, j int) bool) {
	rv := reflect.ValueOf(slice)
	swap := reflect.Swapper(slice)
	stable_func(lessSwap{less, swap}, rv.Len())
}

// SliceIsSorted tests whether a slice is sorted.
//
// The function panics if the provided interface is not a slice.
func SliceIsSorted(slice interface{}, less func(i, j int) bool) bool {
	rv := reflect.ValueOf(slice)
	n := rv.Len()
	for i := n - 1; i > 0; i-- {
		if less(i, i-1) {
			return false
		}
	}
	return true
}
