// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rule

import (
	"net"
	"testing"

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
		{"missing_equals", "pattern=a,bad", true},
		{"empty_key", "=x,pattern=a", true},
		{"unknown_key", "pattern=a,domian=oopsy", true},
		{"duplicate_key", "pattern=a,pattern=b", true},
		{"bad_quote", "pattern='a\\'", true},
		{"ok", "name=r1,pattern='a,b',type=Node,continue=yes", false},
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
			if got["pattern"] != "a,b" {
				t.Fatalf("expected pattern=%q got=%q", "a,b", got["pattern"])
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
		{"missing_pattern", "name=r1,type=Node", true},
		{"type_empty", "pattern=x,type=", true},
		{"type_whitespace", "pattern=x,type=   ", true},
		{"type_separators_only", "pattern=x,type=| |", true},
		{"log_invalid", "log=verbose,pattern=x", true},
		{"continue_invalid", "pattern=x,continue=maybe", true},
		{"domain_append_invalid", "pattern=x,domain_append=maybe", true},
		{"subnet_invalid", "pattern=x,subnet=notacidr", true},
		{"id_and_idset_mutual_exclusion", "pattern=x,id=a,id_set=b", true},
		{"ok_minimal", "pattern=nid{04d}", false},
		{"ok_multi", "name=r1,log=debug,pattern=x,continue=yes,domain_append=on,type=Node| NodeBMC ,subnet=172.16.0.0/24|172.16.1.0/24", false},
		{"id_set_unimplemented", "pattern=x,id_set=x1000s[0-3]c0b0n[0-7]", true},
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
			if r.Action.Pattern == "" {
				t.Fatalf("expected non-empty pattern got=%q", r.Action.Pattern)
			}
			// Ensure type trimming works for the ok_multi case.
			if tt.name == "ok_multi" {
				if r.Match.Types == nil || !r.Match.Types["Node"] || !r.Match.Types["NodeBMC"] {
					t.Fatalf("expected types to include %q and %q got=%v", "Node", "NodeBMC", r.Match.Types)
				}
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
		{"match_all_when_no_match_fields", Rule{Action: Action{Pattern: "x"}}, iiNode, true},
		{"empty_type_map_is_wildcard_matches_empty_type", Rule{Match: Match{Types: map[string]bool{}}, Action: Action{Pattern: "x"}}, iiEmptyType, true},
		{"type_match", Rule{Match: Match{Types: map[string]bool{"Node": true}}, Action: Action{Pattern: "x"}}, iiNode, true},
		{"type_mismatch", Rule{Match: Match{Types: map[string]bool{"Node": true}}, Action: Action{Pattern: "x"}}, iiBMC, false},
		{"subnet_match", Rule{Match: Match{Subnets: []*net.IPNet{mustCIDR(t, "172.16.0.0/24")}}, Action: Action{Pattern: "x"}}, iiNode, true},
		{"id_match_trim", Rule{Match: Match{ID: "  x1000s0c0b0n0  "}, Action: Action{Pattern: "x"}}, iiNode, true},
		{"idset_match", Rule{Match: Match{IDSet: mset}, Action: Action{Pattern: "x"}}, iiNode, true},
		{"compound_all_required", Rule{Match: Match{Types: map[string]bool{"Node": true}, Subnets: []*net.IPNet{mustCIDR(t, "172.16.0.0/24")}, IDSet: mset}, Action: Action{Pattern: "x"}}, iiNode, true},
		{"compound_missing_one", Rule{Match: Match{Types: map[string]bool{"Node": true}, Subnets: []*net.IPNet{mustCIDR(t, "172.16.99.0/24")}, IDSet: mset}, Action: Action{Pattern: "x"}}, iiNode, false},
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

func TestLookupHostname_RuleOrderingContinueAndDomains(t *testing.T) {
	ii := iface.IfaceInfo{CompID: "x1000s0c0b0n0", CompNID: 7, Type: "Node", MAC: "aa", IPList: []net.IP{net.ParseIP("172.16.0.10")}}

	rules := []Rule{
		{Name: "id", Match: Match{ID: "x1000s0c0b0n0"}, Action: Action{Pattern: "special-{id}", Continue: true}},
		{Name: "type", Match: Match{Types: map[string]bool{"Node": true}}, Action: Action{Pattern: "nid{04d}", Domain: "override.local", DomainAppend: false}},
		{Name: "catchall", Action: Action{Pattern: "should-not-win"}},
	}
	got := LookupHostname(nil, ii, "cluster.local", "none", rules)
	if got != "nid0007.override.local" {
		t.Fatalf("expected=%q got=%q", "nid0007.override.local", got)
	}

	rules2 := []Rule{{Name: "type", Match: Match{Types: map[string]bool{"Node": true}}, Action: Action{Pattern: "nid{04d}", Domain: "rack.local", DomainAppend: true}}}
	got2 := LookupHostname(nil, ii, ".cluster.local", "none", rules2)
	if got2 != "nid0007.cluster.local.rack.local" {
		t.Fatalf("expected=%q got=%q", "nid0007.cluster.local.rack.local", got2)
	}

	// domain=none must override domain_append=true and suppress global domain
	rules2b := []Rule{{Name: "none", Match: Match{Types: map[string]bool{"Node": true}}, Action: Action{Pattern: "nid{04d}", Domain: "none", DomainAppend: true}}}
	got2b := LookupHostname(nil, ii, "cluster.local", "none", rules2b)
	if got2b != "nid0007" {
		t.Fatalf("expected=%q got=%q", "nid0007", got2b)
	}

	// domain=none must also override domain_append=true when the *global* domain has a leading dot.
	rules2b2 := []Rule{{Name: "none-dot-global", Match: Match{Types: map[string]bool{"Node": true}}, Action: Action{Pattern: "nid{04d}", Domain: "none", DomainAppend: true}}}
	got2b2 := LookupHostname(nil, ii, ".cluster.local", "none", rules2b2)
	if got2b2 != "nid0007" {
		t.Fatalf("expected=%q got=%q", "nid0007", got2b2)
	}

	// leading dot in rule domain should be trimmed
	rules2c := []Rule{{Name: "dot", Match: Match{Types: map[string]bool{"Node": true}}, Action: Action{Pattern: "nid{04d}", Domain: ".override.local", DomainAppend: false}}}
	got2c := LookupHostname(nil, ii, "cluster.local", "none", rules2c)
	if got2c != "nid0007.override.local" {
		t.Fatalf("expected=%q got=%q", "nid0007.override.local", got2c)
	}

	rules3 := []Rule{{Name: "nope", Match: Match{Types: map[string]bool{"NodeBMC": true}}, Action: Action{Pattern: "bmc{04d}"}}}
	got3 := LookupHostname(nil, ii, "cluster.local", "none", rules3)
	if got3 != "unknown-0007.cluster.local" {
		t.Fatalf("expected=%q got=%q", "unknown-0007.cluster.local", got3)
	}

	// Subnet match only checks the first IP in the list.
	ii2 := iface.IfaceInfo{CompID: "x1000s0c0b0n0", CompNID: 7, Type: "Node", MAC: "aa", IPList: []net.IP{net.ParseIP("172.16.99.10"), net.ParseIP("172.16.0.10")}}
	rules4 := []Rule{{Name: "subnet", Match: Match{Subnets: []*net.IPNet{mustCIDR(t, "172.16.0.0/24")}}, Action: Action{Pattern: "nid{04d}"}}}
	got4 := LookupHostname(nil, ii2, "cluster.local", "none", rules4)
	if got4 != "unknown-0007.cluster.local" {
		t.Fatalf("expected=%q got=%q", "unknown-0007.cluster.local", got4)
	}
}
