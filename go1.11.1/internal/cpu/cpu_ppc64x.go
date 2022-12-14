// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ppc64 || ppc64le
// +build ppc64 ppc64le

package cpu

const CacheLineSize = 128

// ppc64x doesn't have a 'cpuid' equivalent, so we rely on HWCAP/HWCAP2.
// These are linknamed in runtime/os_linux_ppc64x.go and are initialized by
// archauxv().
var hwcap uint
var hwcap2 uint

// HWCAP/HWCAP2 bits. These are exposed by the kernel.
const (
	// ISA Level
	_PPC_FEATURE2_ARCH_2_07 = 0x80000000
	_PPC_FEATURE2_ARCH_3_00 = 0x00800000

	// CPU features
	_PPC_FEATURE_HAS_ALTIVEC     = 0x10000000
	_PPC_FEATURE_HAS_DFP         = 0x00000400
	_PPC_FEATURE_HAS_VSX         = 0x00000080
	_PPC_FEATURE2_HAS_HTM        = 0x40000000
	_PPC_FEATURE2_HAS_ISEL       = 0x08000000
	_PPC_FEATURE2_HAS_VEC_CRYPTO = 0x02000000
	_PPC_FEATURE2_HTM_NOSC       = 0x01000000
	_PPC_FEATURE2_DARN           = 0x00200000
	_PPC_FEATURE2_SCV            = 0x00100000
)

func doinit() {
	options = []option{
		{"htm", &PPC64.HasHTM},
		{"htmnosc", &PPC64.HasHTMNOSC},
		{"darn", &PPC64.HasDARN},
		{"scv", &PPC64.HasSCV},

		// These capabilities should always be enabled on ppc64 and ppc64le:
		//  {"vmx", &PPC64.HasVMX},
		//  {"dfp", &PPC64.HasDFP},
		//  {"vsx", &PPC64.HasVSX},
		//  {"isel", &PPC64.HasISEL},
		//  {"vcrypto", &PPC64.HasVCRYPTO},
	}

	// HWCAP feature bits
	PPC64.HasVMX = isSet(hwcap, _PPC_FEATURE_HAS_ALTIVEC)
	PPC64.HasDFP = isSet(hwcap, _PPC_FEATURE_HAS_DFP)
	PPC64.HasVSX = isSet(hwcap, _PPC_FEATURE_HAS_VSX)

	// HWCAP2 feature bits
	PPC64.IsPOWER8 = isSet(hwcap2, _PPC_FEATURE2_ARCH_2_07)
	PPC64.HasHTM = isSet(hwcap2, _PPC_FEATURE2_HAS_HTM)
	PPC64.HasISEL = isSet(hwcap2, _PPC_FEATURE2_HAS_ISEL)
	PPC64.HasVCRYPTO = isSet(hwcap2, _PPC_FEATURE2_HAS_VEC_CRYPTO)
	PPC64.HasHTMNOSC = isSet(hwcap2, _PPC_FEATURE2_HTM_NOSC)
	PPC64.IsPOWER9 = isSet(hwcap2, _PPC_FEATURE2_ARCH_3_00)
	PPC64.HasDARN = isSet(hwcap2, _PPC_FEATURE2_DARN)
	PPC64.HasSCV = isSet(hwcap2, _PPC_FEATURE2_SCV)
}

func isSet(hwc uint, value uint) bool {
	return hwc&value != 0
}
