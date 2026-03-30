// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package parse

import (
	"strconv"
	"strings"
)

// ParseBoolLoose parses s as a Boolean using synonyms for true or false.
//
// Terms it accepts for true are: 1, true, y, yes, on
// Terms it accepts for false are: 0, false, n, no, off
func ParseBoolLoose(s string) (bool, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "1", "true", "y", "yes", "on":
		return true, nil
	case "0", "false", "n", "no", "off":
		return false, nil
	default:
		// also accept strconv.ParseBool-compatible values
		return strconv.ParseBool(s)
	}
}
