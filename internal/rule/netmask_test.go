// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rule

import (
	"net"
	"testing"
)

func TestCheckValidNetmask_Table(t *testing.T) {
	tests := []struct {
		name string
		m    net.IPMask
		want bool
	}{
		{"/24", net.IPv4Mask(255, 255, 255, 0), true},
		{"/32", net.IPv4Mask(255, 255, 255, 255), true},
		{"/0", net.IPv4Mask(0, 0, 0, 0), true},
		{"non_contiguous", net.IPv4Mask(255, 0, 255, 0), false},
		{"v6_mask_contiguous", net.CIDRMask(64, 128), true},
		{"v6_mask_non_contiguous", net.IPMask([]byte{0xff, 0x00, 0xff, 0x00, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkValidNetmask(tt.m); got != tt.want {
				t.Fatalf("expected=%v got=%v", tt.want, got)
			}
		})
	}
}
