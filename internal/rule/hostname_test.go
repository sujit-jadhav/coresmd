// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rule

import (
	"net"
	"testing"

	"github.com/openchami/coresmd/internal/iface"
)

func TestLookupHostname_DomainAppendModes(t *testing.T) {
	ii := iface.IfaceInfo{CompID: "x1000s0c0b0n0", CompNID: 7, Type: "Node", MAC: "aa", IPList: []net.IP{net.ParseIP("172.16.0.10")}}

	tests := []struct {
		name      string
		globalDom string
		rule      Rule
		want      string
	}{
		{
			name:      "default_global_only",
			globalDom: "cluster.local",
			rule:      Rule{Action: Action{Hostname: "nid{04d}"}},
			want:      "nid0007.cluster.local",
		},
		{
			name:      "default_rule_overrides_global",
			globalDom: "cluster.local",
			rule:      Rule{Action: Action{Hostname: "nid{04d}", Domain: "override.local"}},
			want:      "nid0007.override.local",
		},
		{
			name:      "explicit_global",
			globalDom: "cluster.local",
			rule:      Rule{Action: Action{Hostname: "nid{04d}", Domain: "override.local", DomainAppend: "global"}},
			want:      "nid0007.cluster.local",
		},
		{
			name:      "explicit_rule",
			globalDom: "cluster.local",
			rule:      Rule{Action: Action{Hostname: "nid{04d}", Domain: "override.local", DomainAppend: "rule"}},
			want:      "nid0007.override.local",
		},
		{
			name:      "explicit_global_rule",
			globalDom: "cluster.local",
			rule:      Rule{Action: Action{Hostname: "nid{04d}", Domain: "override.local", DomainAppend: "global|rule"}},
			want:      "nid0007.cluster.local.override.local",
		},
		{
			name:      "explicit_rule_global_order_matters",
			globalDom: "cluster.local",
			rule:      Rule{Action: Action{Hostname: "nid{04d}", Domain: "override.local", DomainAppend: "rule|global"}},
			want:      "nid0007.override.local.cluster.local",
		},
		{
			name:      "explicit_none",
			globalDom: "cluster.local",
			rule:      Rule{Action: Action{Hostname: "nid{04d}", Domain: "override.local", DomainAppend: "none"}},
			want:      "nid0007",
		},
		{
			name:      "explicit_rule_without_rule_domain",
			globalDom: "cluster.local",
			rule:      Rule{Action: Action{Hostname: "nid{04d}", DomainAppend: "rule"}},
			want:      "nid0007",
		},
		{
			name:      "leading_dots_trimmed",
			globalDom: ".cluster.local",
			rule:      Rule{Action: Action{Hostname: "nid{04d}", Domain: ".override.local", DomainAppend: "global|rule"}},
			want:      "nid0007.cluster.local.override.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lookupHostname(tt.rule.Action.Hostname, tt.globalDom, ii, tt.rule)
			if got != tt.want {
				t.Fatalf("expected=%q got=%q", tt.want, got)
			}
		})
	}
}
