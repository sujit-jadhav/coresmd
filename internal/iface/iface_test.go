// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package iface

import (
	"net"
	"testing"

	"github.com/openchami/coresmd/internal/cache"
	"github.com/openchami/coresmd/internal/smdclient"
)

func TestLookupMAC_Table(t *testing.T) {
	mkCache := func() *cache.Cache {
		c := &cache.Cache{}
		c.EthernetInterfaces = map[string]smdclient.EthernetInterface{
			"aa:bb:cc:dd:ee:ff": {MACAddress: "aa:bb:cc:dd:ee:ff", ComponentID: "x0c0s0b0n0", IPAddresses: []smdclient.IPAddress{{IPAddress: "172.16.0.10"}, {IPAddress: "2001:db8::1"}}},
			"11:22:33:44:55:66": {MACAddress: "11:22:33:44:55:66", ComponentID: "x0c0s0b0b0", IPAddresses: []smdclient.IPAddress{{IPAddress: "172.16.10.20"}}},
		}
		c.Components = map[string]smdclient.Component{
			"x0c0s0b0n0": {ID: "x0c0s0b0n0", NID: 7, Type: "Node"},
			"x0c0s0b0b0": {ID: "x0c0s0b0b0", Type: "NodeBMC"},
		}
		return c
	}

	tests := []struct {
		name    string
		mac     string
		cache   *cache.Cache
		wantErr bool
		check   func(t *testing.T, ii IfaceInfo)
	}{
		{name: "nil_cache", mac: "aa:bb:cc:dd:ee:ff", cache: nil, wantErr: true},
		{name: "mac_not_found", mac: "00:00:00:00:00:00", cache: mkCache(), wantErr: true},
		{name: "component_not_found", mac: "aa:bb:cc:dd:ee:ff", cache: func() *cache.Cache { c := mkCache(); delete(c.Components, "x0c0s0b0n0"); return c }(), wantErr: true},
		{
			name:  "node_success_populates_nid_and_ips",
			mac:   "aa:bb:cc:dd:ee:ff",
			cache: mkCache(),
			check: func(t *testing.T, ii IfaceInfo) {
				if ii.Type != "Node" || ii.CompID != "x0c0s0b0n0" {
					t.Fatalf("type/id=%s/%s", ii.Type, ii.CompID)
				}
				if ii.CompNID != 7 {
					t.Fatalf("nid=%d", ii.CompNID)
				}
				if len(ii.IPList) != 2 || !ii.IPList[0].Equal(net.ParseIP("172.16.0.10")) {
					t.Fatalf("ips=%v", ii.IPList)
				}
			},
		},
		{
			name:  "bmc_success_does_not_set_nid",
			mac:   "11:22:33:44:55:66",
			cache: mkCache(),
			check: func(t *testing.T, ii IfaceInfo) {
				if ii.Type != "NodeBMC" {
					t.Fatalf("type=%s", ii.Type)
				}
				if ii.CompNID != 0 {
					t.Fatalf("nid=%d", ii.CompNID)
				}
			},
		},
		{
			name: "no_ips_returns_error",
			mac:  "aa:bb:cc:dd:ee:ff",
			cache: func() *cache.Cache {
				c := mkCache()
				ei := c.EthernetInterfaces["aa:bb:cc:dd:ee:ff"]
				ei.IPAddresses = nil
				c.EthernetInterfaces["aa:bb:cc:dd:ee:ff"] = ei
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ii, err := LookupMAC(nil, tt.mac, tt.cache)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if tt.check != nil {
				tt.check(t, ii)
			}
		})
	}
}
