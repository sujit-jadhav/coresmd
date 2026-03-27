// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rule

import (
	"bytes"
	"net"
	"strings"
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv6"

	"github.com/openchami/coresmd/internal/iface"
)

// staticSet is a local IDSetMatcher implementation used for tests.
// It allows exercising Match.IDSet behavior without depending on CompileIDSet().
type staticSet map[string]bool

func (s staticSet) Match(id string) bool { return s[id] }

func (s staticSet) String() string { return "staticSet" }

func mustCIDR(t *testing.T, s string) *net.IPNet {
	t.Helper()
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		t.Fatalf("ParseCIDR(%q): %v", s, err)
	}
	return n
}

func TestCreateRuleCompDict_Table(t *testing.T) {
	tests := []struct {
		name    string
		rule    string
		wantErr bool
	}{
		{"empty", " ", true},
		{"missing_colon", "hostname:a,bad", true},
		{"empty_key", ":x,hostname:a", true},
		{"unknown_key", "hostname:a,domian:oopsy", true},
		{"duplicate_key", "hostname:a,hostname:b", true},
		{"bad_quote", "hostname:'a\\'", true},
		{"ok", "name:r1,hostname:'a,b',type:Node,continue:yes,routers:192.0.2.1|192.0.2.2", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createRuleCompDict(tt.rule)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got["hostname"] != "a,b" {
				t.Fatalf("expected hostname=%q got=%q", "a,b", got["hostname"])
			}
		})
	}
}

func TestParseRule_Table(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"missing_actions", "name:r1,type:Node", true},
		{"routers_only_ok", "routers:192.0.2.1", false},
		{"routers_bad_ip", "routers:not_an_ip", true},
		{"type_empty", "hostname:x,type:", true},
		{"type_whitespace", "hostname:x,type:   ", true},
		{"type_separators_only", "hostname:x,type:| |", true},
		{"log_invalid", "log:verbose,hostname:x", true},
		{"continue_invalid", "hostname:x,continue:maybe", true},
		{"domain_append_invalid", "hostname:x,domain_append:maybe", true},
		{"domain_append_none_combo_invalid", "hostname:x,domain_append:none|rule", true},
		{"domain_append_duplicate_global", "hostname:x,domain:override.local,domain_append:global|global", true},
		{"domain_append_duplicate_rule", "hostname:x,domain:override.local,domain_append:rule|rule", true},
		{"domain_append_duplicate_mixed", "hostname:x,domain:override.local,domain_append:global|rule|global", true},
		{"domain_none_removed", "hostname:x,domain:none", true},
		{"subnet_invalid", "hostname:x,subnet:notacidr", true},
		{"id_and_idset_mutual_exclusion", "hostname:x,id:a,id_set:b", true},
		{"ok_minimal", "hostname:nid{04d}", false},
		{"ok_multi", "name:r1,log:debug,hostname:x,continue:yes,domain_append:global|rule,type:Node| NodeBMC ,subnet:172.16.0.0/24|172.16.1.0/24", false},
		{"ok_domain_append_rule_global", "hostname:x,domain:override.local,domain_append:rule|global", false},
		{"id_set_unimplemented", "hostname:x,id_set:x1000s[0-3]c0b0n[0-7]", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := ParseRule(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			// ParseRule() does not know the global rule_log value. If 'log' is
			// omitted, it should be left empty for the caller to apply defaults.
			if tt.name == "ok_minimal" || tt.name == "routers_only_ok" {
				if r.Log != "" {
					t.Fatalf("expected default rule log to be empty got=%q", r.Log)
				}
			}
			if tt.name != "routers_only_ok" && r.Action.Hostname == "" {
				t.Fatalf("expected non-empty hostname got=%q", r.Action.Hostname)
			}
			if tt.name == "routers_only_ok" {
				if len(r.Action.Routers) != 1 {
					t.Fatalf("expected 1 router got=%d", len(r.Action.Routers))
				}
			}
			// Ensure type trimming works for the ok_multi case.
			if tt.name == "ok_multi" {
				if r.Match.Types == nil || !r.Match.Types["Node"] || !r.Match.Types["NodeBMC"] {
					t.Fatalf("expected types to include %q and %q got=%v", "Node", "NodeBMC", r.Match.Types)
				}
				if r.Action.DomainAppend != "global|rule" {
					t.Fatalf("expected domain_append=%q got=%q", "global|rule", r.Action.DomainAppend)
				}
			}
			if tt.name == "ok_domain_append_rule_global" {
				if r.Action.DomainAppend != "rule|global" {
					t.Fatalf("expected domain_append=%q got=%q", "rule|global", r.Action.DomainAppend)
				}
			}
		})
	}
}

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

func TestRuleMatchIface_Combinations(t *testing.T) {
	iiNode := iface.IfaceInfo{CompID: "x1000s0c0b0n0", CompNID: 7, Type: "Node", MAC: "aa", IPList: []net.IP{net.ParseIP("172.16.0.10")}}
	iiBMC := iface.IfaceInfo{CompID: "x1000s0c0b0n0", Type: "NodeBMC", MAC: "bb", IPList: []net.IP{net.ParseIP("172.16.10.10")}}
	iiEmptyType := iface.IfaceInfo{CompID: "x1000s0c0b0n0", CompNID: 7, Type: "", MAC: "cc", IPList: []net.IP{net.ParseIP("172.16.0.10")}}

	mset := staticSet{"x1000s0c0b0n0": true, "x1000s0c0b0n1": true}

	tests := []struct {
		name      string
		rule      Rule
		ii        iface.IfaceInfo
		wantMatch bool
	}{
		{"match_all_when_no_match_fields", Rule{Action: Action{Hostname: "x"}}, iiNode, true},
		{"empty_type_map_is_wildcard_matches_empty_type", Rule{Match: Match{Types: map[string]bool{}}, Action: Action{Hostname: "x"}}, iiEmptyType, true},
		{"type_match", Rule{Match: Match{Types: map[string]bool{"Node": true}}, Action: Action{Hostname: "x"}}, iiNode, true},
		{"type_mismatch", Rule{Match: Match{Types: map[string]bool{"Node": true}}, Action: Action{Hostname: "x"}}, iiBMC, false},
		{"subnet_match", Rule{Match: Match{Subnets: []*net.IPNet{mustCIDR(t, "172.16.0.0/24")}}, Action: Action{Hostname: "x"}}, iiNode, true},
		{"id_match_trim", Rule{Match: Match{ID: "  x1000s0c0b0n0  "}, Action: Action{Hostname: "x"}}, iiNode, true},
		{"idset_match", Rule{Match: Match{IDSet: mset}, Action: Action{Hostname: "x"}}, iiNode, true},
		{"compound_all_required", Rule{Match: Match{Types: map[string]bool{"Node": true}, Subnets: []*net.IPNet{mustCIDR(t, "172.16.0.0/24")}, IDSet: mset}, Action: Action{Hostname: "x"}}, iiNode, true},
		{"compound_missing_one", Rule{Match: Match{Types: map[string]bool{"Node": true}, Subnets: []*net.IPNet{mustCIDR(t, "172.16.99.0/24")}, IDSet: mset}, Action: Action{Hostname: "x"}}, iiNode, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := tt.rule.MatchIface(tt.ii)
			if m != tt.wantMatch {
				t.Fatalf("expected match=%v got=%v", tt.wantMatch, m)
			}
		})
	}
}

func TestEvaluate4_HostnameRoutersAndDefault(t *testing.T) {
	ii := iface.IfaceInfo{CompID: "x1000s0c0b0n0", CompNID: 7, Type: "Node", MAC: "aa", IPList: []net.IP{net.ParseIP("172.16.0.10")}}

	// Matching rules set hostname and routers; default should NOT override.
	resp, err := dhcpv4.New()
	if err != nil {
		t.Fatalf("unexpected error creating dhcpv4 message: %v", err)
	}
	rules := []Rule{{
		Name:   "node",
		Match:  Match{Types: map[string]bool{"Node": true}},
		Action: Action{Hostname: "nid{04d}", Domain: "override.local", Routers: []net.IP{net.ParseIP("192.0.2.1"), net.ParseIP("192.0.2.2")}},
	}}
	Evaluate4(nil, ii, "cluster.local", "none", resp, rules)

	if got := string(bytes.Trim(resp.Options.Get(dhcpv4.OptionHostName), "\x00")); got != "nid0007.override.local" {
		t.Fatalf("expected=%q got=%q", "nid0007.override.local", got)
	}
	if got := resp.Options.Get(dhcpv4.OptionRouter); len(got) != 8 {
		t.Fatalf("expected %d bytes of router option got=%d", 8, len(got))
	}

	// Routers-only rule is allowed; hostname falls back to DefaultPattern.
	resp2, err := dhcpv4.New()
	if err != nil {
		t.Fatalf("unexpected error creating dhcpv4 message: %v", err)
	}
	rules2 := []Rule{{Name: "rtrs", Action: Action{Routers: []net.IP{net.ParseIP("192.0.2.1")}}}}
	Evaluate4(nil, ii, "cluster.local", "none", resp2, rules2)
	if got := string(bytes.Trim(resp2.Options.Get(dhcpv4.OptionHostName), "\x00")); got != "unknown-0007.cluster.local" {
		t.Fatalf("expected=%q got=%q", "unknown-0007.cluster.local", got)
	}

	// No rules match => default hostname applied.
	resp3, err := dhcpv4.New()
	if err != nil {
		t.Fatalf("unexpected error creating dhcpv4 message: %v", err)
	}
	rules3 := []Rule{{Name: "nope", Match: Match{Types: map[string]bool{"NodeBMC": true}}, Action: Action{Hostname: "bmc{04d}"}}}
	Evaluate4(nil, ii, "cluster.local", "none", resp3, rules3)
	if got := string(bytes.Trim(resp3.Options.Get(dhcpv4.OptionHostName), "\x00")); got != "unknown-0007.cluster.local" {
		t.Fatalf("expected=%q got=%q", "unknown-0007.cluster.local", got)
	}

	// Subnet match only checks the first IP in the list.
	ii2 := iface.IfaceInfo{CompID: "x1000s0c0b0n0", CompNID: 7, Type: "Node", MAC: "aa", IPList: []net.IP{net.ParseIP("172.16.99.10"), net.ParseIP("172.16.0.10")}}
	resp4, err := dhcpv4.New()
	if err != nil {
		t.Fatalf("unexpected error creating dhcpv4 message: %v", err)
	}
	rules4 := []Rule{{Name: "subnet", Match: Match{Subnets: []*net.IPNet{mustCIDR(t, "172.16.0.0/24")}}, Action: Action{Hostname: "nid{04d}"}}}
	Evaluate4(nil, ii2, "cluster.local", "none", resp4, rules4)
	if got := string(bytes.Trim(resp4.Options.Get(dhcpv4.OptionHostName), "\x00")); got != "unknown-0007.cluster.local" {
		t.Fatalf("expected=%q got=%q", "unknown-0007.cluster.local", got)
	}
}

func TestEvaluate6_HostnameAndDefault(t *testing.T) {
	ii := iface.IfaceInfo{CompID: "x1000s0c0b0n0", CompNID: 7, Type: "Node", MAC: "aa", IPList: []net.IP{net.ParseIP("172.16.0.10")}}

	resp, err := dhcpv6.NewMessage()
	if err != nil {
		t.Fatalf("unexpected error creating dhcpv6 message: %v", err)
	}
	rules := []Rule{{Name: "node", Match: Match{Types: map[string]bool{"Node": true}}, Action: Action{Hostname: "nid{04d}", Domain: "override.local"}}}
	Evaluate6(nil, ii, "cluster.local", "none", resp, rules)

	opt := resp.GetOneOption(dhcpv6.OptionFQDN)
	if opt == nil {
		t.Fatalf("expected FQDN option to be set got=nil")
	}
	fqdn, ok := opt.(*dhcpv6.OptFQDN)
	if !ok {
		t.Fatalf("expected OptFQDN got=%T", opt)
	}
	got := strings.Join(fqdn.DomainName.Labels, ".")
	if got != "nid0007.override.local" {
		t.Fatalf("expected=%q got=%q", "nid0007.override.local", got)
	}

	// No match => default hostname applied.
	resp2, err := dhcpv6.NewMessage()
	if err != nil {
		t.Fatalf("unexpected error creating dhcpv6 message: %v", err)
	}
	rules2 := []Rule{{Name: "nope", Match: Match{Types: map[string]bool{"NodeBMC": true}}, Action: Action{Hostname: "bmc{04d}"}}}
	Evaluate6(nil, ii, "cluster.local", "none", resp2, rules2)
	opt2 := resp2.GetOneOption(dhcpv6.OptionFQDN)
	if opt2 == nil {
		t.Fatalf("expected FQDN option to be set got=nil")
	}
	fqdn2 := opt2.(*dhcpv6.OptFQDN)
	got2 := strings.Join(fqdn2.DomainName.Labels, ".")
	if got2 != "unknown-0007.cluster.local" {
		t.Fatalf("expected=%q got=%q", "unknown-0007.cluster.local", got2)
	}
}
