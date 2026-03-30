// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package parse

import (
	"strings"
	"unicode"
)

// Fields splits s into fields separated by any Unicode whitespace. Consecutive
// whitespace is treated as a single separator, so no empty strings are
// returned. It is equivalent to strings.Fields, but implemented explicitly
// using rune-wise iteration.
func Fields(s string) []string {
	var out []string
	var cur strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}
