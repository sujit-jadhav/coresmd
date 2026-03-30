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

	// Gather and trim leading dot off global and rule domains
	globalDom := strings.TrimLeft(domain, ".")
	ruleDom := strings.TrimSpace(rule.Action.Domain)
	if ruleDom != "" {
		ruleDom = strings.TrimLeft(ruleDom, ".")
	}

	// Don't attempt to append any domain if told not to
	if strings.EqualFold(strings.TrimSpace(rule.Action.Domain), "none") {
		return hname
	}

	mode := strings.TrimSpace(strings.ToLower(rule.Action.DomainAppend))
	if mode == "none" {
		return hname
	}

	// Default behavior when domain_append is omitted
	if mode == "" {
		if ruleDom != "" {
			// Rule domain overrides global domain
			return fmt.Sprintf("%s.%s", hname, ruleDom)
		}
		if globalDom != "" {
			// Fallback to global domain
			return fmt.Sprintf("%s.%s", hname, globalDom)
		}
		return hname
	}

	// Explicit behavior.
	labels := []string{hname}
	for _, tok := range strings.Split(mode, "|") {
		tok = strings.TrimSpace(tok)
		switch tok {
		case "global":
			if globalDom != "" {
				labels = append(labels, globalDom)
			}
		case "rule":
			if ruleDom != "" {
				labels = append(labels, ruleDom)
			}
		}
	}
	return strings.Join(labels, ".")
}
