// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package parse

import (
	"errors"
	"strings"
)

// Unquote removes surrounding single/double quotes if present,
// and interprets simple escapes like \" and \'.
func Unquote(v string) (string, error) {
	v = strings.TrimSpace(v)
	if len(v) < 2 {
		return v, nil
	}
	first := v[0]
	last := v[len(v)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		body := v[1 : len(v)-1]

		// Interpret backslash escapes minimally; keep it conservative.
		var b strings.Builder
		esc := false
		for i := 0; i < len(body); i++ {
			c := body[i]
			if esc {
				// allow escaping quote and backslash; otherwise keep literal char
				switch c {
				case '\\', '"', '\'':
					b.WriteByte(c)
				case 'n':
					b.WriteByte('\n')
				case 't':
					b.WriteByte('\t')
				default:
					b.WriteByte(c)
				}
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			b.WriteByte(c)
		}
		if esc {
			return "", errors.New("unterminated escape in quoted string")
		}
		return b.String(), nil
	}
	return v, nil
}
