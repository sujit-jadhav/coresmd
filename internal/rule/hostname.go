// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rule

import (
	"fmt"
	"strings"

	"github.com/openchami/coresmd/internal/hostname"
	"github.com/openchami/coresmd/internal/iface"
)

func lookupHostname(pattern, domain string, ii iface.IfaceInfo, rule Rule) (hname string) {
	// Compile hostname from pattern
	hname = hostname.ExpandHostnamePattern(pattern, ii.CompNID, ii.CompID)

	// Trim global domain, if set
	if dom := strings.TrimSpace(domain); dom != "" {
		domain = dom
	} else {
		domain = ""
	}

	// Handle domain setting
	if rdom := strings.TrimSpace(rule.Action.Domain); rdom != "" {
		// domain=none always suppresses any domain behavior (including domain_append).
		if rdom == "none" {
			return hname
		}
		rdom = strings.TrimLeft(rdom, ".")
		if rule.Action.DomainAppend {
			// Rule domain specified to append, append rule domain to hname
			// appended with global domain (if set).
			if domain != "" {
				hname = fmt.Sprintf("%s.%s.%s", hname, strings.TrimLeft(domain, "."), strings.TrimLeft(rdom, "."))
			} else {
				// Append specified, but global domain was not set. Use only
				// rule domain.
				hname = fmt.Sprintf("%s.%s", hname, strings.TrimLeft(rdom, "."))
			}
		} else {
			// Rule domain specified to replace global domain, do so.
			hname = fmt.Sprintf("%s.%s", hname, strings.TrimLeft(rdom, "."))
		}
	} else {
		// Rule domain was not specified, DomainAppend setting doesn't matter.
		// Append the global domain only if set.
		if domain != "" {
			hname = fmt.Sprintf("%s.%s", hname, strings.TrimLeft(domain, "."))
		}
	}

	return
}
