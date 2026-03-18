// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package ipxe

import (
	"net"
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/iana"
)

func TestServeIPXEBootloader_Table(t *testing.T) {
	mkReq := func(arch iana.Arch, include bool) *dhcpv4.DHCPv4 {
		req := &dhcpv4.DHCPv4{ClientHWAddr: net.HardwareAddr{0, 1, 2, 3, 4, 5}, Options: dhcpv4.Options{}}
		if include {
			req.Options.Update(dhcpv4.OptClientArch(arch))
		}
		return req
	}
	mkResp := func() *dhcpv4.DHCPv4 { return &dhcpv4.DHCPv4{Options: dhcpv4.Options{}} }

	tests := []struct {
		name        string
		req         *dhcpv4.DHCPv4
		wantHandled bool
		wantBoot    string
	}{
		{"no_arch_option", mkReq(0, false), false, ""},
		{"intel_x86", mkReq(iana.INTEL_X86PC, true), true, "undionly.kpxe"},
		{"efi_ia32", mkReq(iana.EFI_IA32, true), true, "ipxe-i386.efi"},
		{"efi_x86_64", mkReq(iana.EFI_X86_64, true), true, "ipxe-x86_64.efi"},
		{"efi_arm32", mkReq(iana.EFI_ARM32, true), true, "ipxe-arm32.efi"},
		{"efi_arm64", mkReq(iana.EFI_ARM64, true), true, "ipxe-arm64.efi"},
		{"unknown_arch", mkReq(iana.Arch(999), true), false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, handled := ServeIPXEBootloader(nil, tt.req, mkResp())
			if handled != tt.wantHandled {
				t.Fatalf("handled=%v want %v", handled, tt.wantHandled)
			}
			if !tt.wantHandled {
				return
			}
			got := string(resp.Options.Get(dhcpv4.OptionBootfileName))
			if got != tt.wantBoot {
				t.Fatalf("bootfile=%q want %q", got, tt.wantBoot)
			}
		})
	}
}
