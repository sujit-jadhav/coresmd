// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rule

import "testing"

func TestErrors_Table(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"keyval_format", NewErrKeyValFormat(3, "bogus"), "element 3: expected key:val, got \"bogus\""},
		{"no_key", NewErrNoKey(2, ":x"), "element 2: empty key (got \":x\")"},
		{"unknown_key", NewErrUnknownKey(1, "domian"), "element 1: unknown key \"domian\""},
		{"duplicate_key", NewErrDuplicateKey(4, "hostname:x", "hostname"), "element 4: duplicate key \"hostname\": got \"hostname:x\""},
		{"required_keys", NewErrRequiredKeys("hostname", "routers"), "required key missing, at least one of [hostname routers]"},
		{"invalid_value", NewErrInvalidValue("domain_append", "maybe", "global|rule|none"), "invalid value for key \"domain_append\" (expected global|rule|none but got \"maybe\")"},
		{"mutual_exclusion", NewErrMutualExclusion("netmask", "cidr"), "keys are mutually exclusive: [netmask cidr]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("expected=%q got=%q", tt.want, got)
			}
		})
	}
}
