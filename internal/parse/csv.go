// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package parse

import (
	"errors"
	"strings"
)

// SplitCSV splits out comma-separated values of a string into a string array.
// Commas used for splitting are NOT within single or double quotes.
//
// Examples:
//   - "k=v,k=v,pattern='a,b',k=v" -> "k=v","k=v","pattern='a,b'","k=v"
func SplitCSV(s string) ([]string, error) {
	var out []string
	var cur strings.Builder
	var quote rune // 0, '\'', '"'
	escaped := false

	for _, r := range s {
		if escaped {
			// keep escape sequences
			// Unquote() can be used to remove a layer of quotes
			cur.WriteRune(r)
			escaped = false
			continue
		}
		if quote != 0 {
			// in quotes
			switch r {
			case '\\':
				escaped = true
				cur.WriteRune(r)
			case quote:
				quote = 0
				cur.WriteRune(r)
			default:
				cur.WriteRune(r)
			}
			continue
		}

		// not in quotes
		switch r {
		case '\'', '"':
			quote = r
			cur.WriteRune(r)
		case ',':
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}

	if quote != 0 {
		return nil, errors.New("unterminated quote")
	}
	if escaped {
		// trailing backslash is assumed to be an error
		return nil, errors.New("trailing escape")
	}

	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out, nil
}
