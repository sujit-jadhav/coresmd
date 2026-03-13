// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package parse

import "testing"

func TestSplitCSV_Table(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    []string
		wantErr bool
	}{
		{"simple", "a=b,c=d", []string{"a=b", "c=d"}, false},
		{"quoted_commas_single", "a=b,pattern='x,y',c=d", []string{"a=b", "pattern='x,y'", "c=d"}, false},
		{"quoted_commas_double", "a=b,pattern=\"x,y\",c=d", []string{"a=b", "pattern=\"x,y\"", "c=d"}, false},
		{"escaped_quote_inside", "pattern='x\\'y',a=b", []string{"pattern='x\\'y'", "a=b"}, false},
		{"unterminated_quote", "a=b,pattern='x,y", nil, true},
		{"trailing_escape", "a=b,pattern='x\\'", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitCSV(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len=%d want %d got=%v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("[%d]=%q want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestUnquote_Table(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"no_quotes", "abc", "abc", false},
		{"single_quotes", "'abc'", "abc", false},
		{"double_quotes", "\"abc\"", "abc", false},
		{"escapes", "'a\\n\\t\\'b'", "a\n\t'b", false},
		{"unterminated_escape", "'a\\'", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Unquote(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("got=%q want %q", got, tt.want)
			}
		})
	}
}

func TestParseBoolLoose_Table(t *testing.T) {
	tests := []struct {
		in      string
		want    bool
		wantErr bool
	}{
		{"true", true, false},
		{"TRUE", true, false},
		{"1", true, false},
		{"yes", true, false},
		{"on", true, false},
		{"false", false, false},
		{"0", false, false},
		{"no", false, false},
		{"off", false, false},
		{"t", true, false},
		{"f", false, false},
		{"notabool", false, true},
	}
	for _, tt := range tests {
		got, err := ParseBoolLoose(tt.in)
		if (err != nil) != tt.wantErr {
			t.Fatalf("%q err=%v wantErr=%v", tt.in, err, tt.wantErr)
		}
		if tt.wantErr {
			continue
		}
		if got != tt.want {
			t.Fatalf("%q got=%v want %v", tt.in, got, tt.want)
		}
	}
}

func TestFields(t *testing.T) {
	in := "\t a  b\n\r\n c  "
	got := Fields(in)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d got=%v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("[%d]=%q want %q", i, got[i], want[i])
		}
	}
}
