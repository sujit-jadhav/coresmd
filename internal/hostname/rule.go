// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package hostname

import (
	"crypto/sha256"
	"fmt"
	"net"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/openchami/coresmd/internal/iface"
	"github.com/openchami/coresmd/internal/parse"
)

const (
	DefaultPattern = "unknown-{04d}"
)

// Rule represents a hostname rule
type Rule struct {
	Name   string // ID of rule
	Log    string // "info" (log match), "debug" (log match/skip), "none"/omit (don't log)
	Match  Match  // criteria for rule to match host
	Action Action // what to do if rule matches host
}

func (r Rule) String() string {
	return fmt.Sprintf("name=%s,log=%s,%s,%s",
		r.Name,
		r.Log,
		r.Match,
		r.Action,
	)
}

// MatchIface evaluates the rule for the passed interface and determines whether
// the rule matches and whether the evaluation should continue down the chain.
func (r Rule) MatchIface(ii iface.IfaceInfo) (matches, cont bool) {
	// Determine whether to continue
	cont = r.Action.Continue

	matchSet := make(map[string]bool)
	if r.Match.Types != nil {
		matchSet["types"] = true
	}
	if len(r.Match.Subnets) > 0 {
		matchSet["subnets"] = true
	}
	if id := strings.TrimSpace(r.Match.ID); id != "" {
		matchSet["id"] = true
		r.Match.ID = id
	} else if r.Match.IDSet != nil {
		matchSet["id_set"] = true
	}

	matchCounter := 0

	// Match type
	if _, ok := matchSet["types"]; ok {
		if _, ok := r.Match.Types[ii.Type]; ok {
			matchCounter++
		}
	}

	// Match subnet
	if _, ok := matchSet["subnets"]; ok && len(ii.IPList) > 0 {
		for _, subnet := range r.Match.Subnets {
			if subnet == nil {
				continue
			}
			// We assume that the first IP will be assigned
			if subnet.Contains(ii.IPList[0]) {
				matchCounter++
				break
			}
		}
	}

	if _, ok := matchSet["id"]; ok {
		// Match ID
		if ii.CompID == r.Match.ID {
			matchCounter++
		}
	} else if _, ok := matchSet["id_set"]; ok {
		// Match ID set
		if r.Match.IDSet.Match(ii.CompID) {
			matchCounter++
		}
	}

	// Tally up the matches, returning true if everything matches
	if matchCounter == len(matchSet) {
		matches = true
	}

	return
}

// Match represents criteria for hostname rule to match host
type Match struct {
	Types   map[string]bool // any type in map matches, all types if empty
	Subnets []*net.IPNet    // any subnet in slice matches, all subnets if empty
	ID      string          // xname to match, any if empty
	IDSet   IDSetMatcher    // set of xnames to match, any if nil
}

func (m Match) String() string {
	var matchStr string

	if m.Types != nil {
		var notfirst bool
		typStr := ",types="
		for typ := range m.Types {
			if notfirst {
				typStr += fmt.Sprintf("|%s", typ)
			} else {
				typStr += fmt.Sprintf("%s", typ)
				notfirst = true
			}
		}
		matchStr += typStr
	}

	if m.Subnets != nil {
		var notfirst bool
		subnetStr := ",subnets="
		for _, subnet := range m.Subnets {
			if notfirst {
				subnetStr += fmt.Sprintf("|%s", subnet)
			} else {
				subnetStr += fmt.Sprintf("%s", subnet)
				notfirst = true
			}
		}
		matchStr += subnetStr
	}

	if id := strings.TrimSpace(m.ID); id != "" {
		matchStr += fmt.Sprintf(",id=%s", id)
	}

	if m.IDSet != nil {
		matchStr += fmt.Sprintf(",id_set=%s", m.IDSet)
	}

	return strings.TrimLeft(matchStr, ",")
}

// Action represents an action to take upon a rule matching
type Action struct {
	Pattern      string // hostname pattern to apply
	Domain       string // domain to append to hostname, overrides global domain, "none" ignores
	DomainAppend bool   // whether to append domain or override it
	Continue     bool   // whether to continue parsing subsequent rules if this matches
}

func (a Action) String() string {
	var actionStr string

	pat := strings.TrimSpace(a.Pattern)
	if pat != "" {
		actionStr += fmt.Sprintf(",pattern=%s", pat)
	}

	dom := strings.TrimSpace(a.Domain)
	if dom != "" {
		actionStr += fmt.Sprintf(",domain=%s", dom)
	}

	actionStr += fmt.Sprintf(",domain_append=%v", a.DomainAppend)
	actionStr += fmt.Sprintf(",continue=%v", a.Continue)

	return strings.TrimLeft(actionStr, ",")
}

// IDSetMatcher provides a function for matching a component ID to a defined set
// of IDs.
type IDSetMatcher interface {
	Match(id string) bool
	String() string // for debugging purposes
}

// CompileIDSet creates an IDSetMatcher from a string expression representing
// the set of IDs.
func CompileIDSet(expr string) (IDSetMatcher, error) {
	// TODO: implement
	return nil, fmt.Errorf("CompileIDSet() is not implemented: %q", expr)
}

// ParseRule parses a string representing a hostname rule (everything to the
// right of 'hostname_rule=') and returns a representative Rule.
func ParseRule(rule string) (Rule, error) {
	comps, err := createRuleCompDict(rule)
	if err != nil {
		return Rule{}, err
	}

	var (
		r Rule
		m Match
		a Action
	)

	// log (optional)
	if log, ok := comps["log"]; ok && log != "" {
		switch log {
		case "info", "debug", "none":
			r.Log = log
		default:
			return Rule{}, NewErrInvalidValue("log", log, "'info', 'debug', or 'none'")
		}
	}

	// name, rule identifier (generated below if not set)
	if name, ok := comps["name"]; ok && name != "" {
		r.Name = name
	}

	// pattern (action)
	if pat, ok := comps["pattern"]; ok && pat != "" {
		a.Pattern = pat
	} else {
		return Rule{}, NewErrRequiredKey("pattern")
	}

	// domain override (optional)
	if dom, ok := comps["domain"]; ok && dom != "" {
		a.Domain = dom
	}

	// continue (optional)
	if cont, ok := comps["continue"]; ok && cont != "" {
		if b, err := parse.ParseBoolLoose(cont); err != nil {
			return Rule{}, NewErrInvalidValue("continue", cont, "boolean")
		} else {
			a.Continue = b
		}
	}

	// domain_append (optional)
	if domapp, ok := comps["domain_append"]; ok && domapp != "" {
		if b, err := parse.ParseBoolLoose(domapp); err != nil {
			return Rule{}, NewErrInvalidValue("domain_append", domapp, "boolean")
		} else {
			a.Continue = b
		}
	}

	// match by type (optional; multivalue)
	//
	// Examples:
	//  - type=Node                    # single type
	//  - type=Node|NodeBMC|HSNSwitch  # multiple types
	if matchType, ok := comps["type"]; ok && matchType != "" {
		m.Types = make(map[string]bool)
		for _, t := range strings.Split(matchType, "|") {
			if t == "" {
				continue
			}
			m.Types[t] = true
		}
	}

	// match by subnet (optional; multivalue)
	//
	// Examples:
	//  - subnet=172.16.0.0/24                # single subnet
	//  - subnet=172.16.0.0/24|172.16.1.0/21  # multiple subnets
	if matchSubnet, ok := comps["subnet"]; ok && matchSubnet != "" {
		for _, s := range strings.Split(matchSubnet, "|") {
			if _, ipnet, err := net.ParseCIDR(strings.TrimSpace(s)); err != nil {
				return Rule{}, NewErrInvalidValue("subnet", s, "network subnet (e.g. 172.16.0.0/21)")
			} else {
				m.Subnets = append(m.Subnets, ipnet)
			}
		}
	}

	// match by single ID (optional)
	// mutually exclusive with matching by ID set
	if id, ok := comps["id"]; ok && id != "" {
		m.ID = id
	}

	// match by ID set (optional)
	// mutually exclusive with matching by ID
	if idset, ok := comps["id_set"]; ok && idset != "" {
		if matcher, err := CompileIDSet(idset); err != nil {
			return Rule{}, NewErrInvalidValue("id_set", idset, "valid ID set (e.g. x1000s0c0b[0-3]n0)")
		} else {
			m.IDSet = matcher
		}
	}

	// ensure id and id_set mutual exclusion
	if m.ID != "" && m.IDSet != nil {
		return Rule{}, NewErrMutualExclusion("id", "id_set")
	}

	r.Match = m
	r.Action = a

	// Generate rule name if it wasn't set
	if n := strings.TrimSpace(r.Name); n == "" {
		sum := sha256.Sum256([]byte(r.String()))
		r.Name = fmt.Sprintf("rule-%08x", sum[:4])
	}

	return r, nil
}

// LookupHostname takes interface information and a list of hostname rules and
// returns the hostname resulting from evaluating the rules in the list. It also
// takes a logger so rules can be logged, if enabled.
func LookupHostname(logger *logrus.Entry, ii iface.IfaceInfo, domain, hlog string, hrules []Rule) (hostname string) {
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}
	logRejection := func(idx int, rule Rule) {
		var rejectEnabled bool
		switch hlog {
		case "debug":
			rejectEnabled = true
		case "", "info", "none":
			// Do nothing
		default:
			err := NewErrInvalidValue("hostname_log", hlog, "'debug', 'info', or 'none'")
			logger.Error(err)
		}
		switch rule.Log {
		case "debug":
			rejectEnabled = true
		case "", "info", "none":
			// Do nothing
		default:
			err := NewErrInvalidValue("log", rule.Log, "'debug', 'info', or 'none'")
			logger.Errorf("rule[%d] (%s) %v", idx, rule.Name, err)
		}
		if rejectEnabled {
			logger.WithFields(logrus.Fields{
				"comp_id":   ii.CompID,
				"comp_nid":  ii.CompNID,
				"comp_type": ii.Type,
				"mac":       ii.MAC,
				"ips":       ii.IPList,
			}).Infof("rule[%d] (%s) did not match", idx, rule.Name)
		}
	}
	logMatch := func(idx int, rule Rule) {
		var matchEnabled bool
		switch hlog {
		case "debug", "info":
			matchEnabled = true
		case "", "none":
			// Do nothing
		default:
			err := NewErrInvalidValue("hostname_log", hlog, "'debug', 'info', or 'none'")
			logger.Error(err)
		}
		switch rule.Log {
		case "debug":
			matchEnabled = true
		case "", "info", "none":
			// Do nothing
		default:
			err := NewErrInvalidValue("log", rule.Log, "'debug', 'info', or 'none'")
			logger.Errorf("rule[%d] (%s) %v", idx, rule.Name, err)
		}
		if matchEnabled {
			logger.WithFields(logrus.Fields{
				"comp_id":   ii.CompID,
				"comp_nid":  ii.CompNID,
				"comp_type": ii.Type,
				"mac":       ii.MAC,
				"ips":       ii.IPList,
			}).Infof("rule[%d] (%s) matched", idx, rule.Name)
		}
	}

	pattern := DefaultPattern
	for idx, rule := range hrules {
		matches, cont := rule.MatchIface(ii)
		if matches {
			logMatch(idx, rule)

			// Rule matches, so set the pattern.
			if pat := strings.TrimSpace(rule.Action.Pattern); pat != "" {
				pattern = pat
			}

			setHostname(pattern, domain, ii, rule)

			if !cont {
				// Continue not specified for match, so stop here.
				break
			}
		} else {
			// Continue only matters for matching, so not matching means we
			// continue searching rules until either one matches or the rule
			// list is exhausted.
			logRejection(idx, rule)
		}
	}

	return
}

func setHostname(pattern, domain string, ii iface.IfaceInfo, rule Rule) (hostname string) {
	// Compile hostname from pattern
	hostname = ExpandHostnamePattern(pattern, ii.CompNID, ii.CompID)

	// Trim global domain, if set
	if dom := strings.TrimSpace(domain); dom != "" {
		domain = dom
	} else {
		domain = ""
	}

	// Handle domain setting
	if rdom := strings.TrimSpace(rule.Action.Domain); rdom != "" {
		if rule.Action.DomainAppend {
			// Rule domain specified to append, append rule domain to hostname
			// appended with global domain (if set).
			if domain != "" {
				hostname = fmt.Sprintf("%s.%s.%s", hostname, strings.TrimLeft(domain, "."), strings.TrimLeft(rdom, "."))
			} else {
				// Append specified, but global domain was not set. Use only
				// rule domain.
				hostname = fmt.Sprintf("%s.%s", hostname, strings.TrimLeft(rdom, "."))
			}
		} else {
			// Rule domain specified to replace global domain, do so.
			hostname = fmt.Sprintf("%s.%s", hostname, strings.TrimLeft(rdom, "."))
		}
	} else {
		// Rule domain was not specified, DomainAppend setting doesn't matter.
		// Append the global domain only if set.
		if domain != "" {
			hostname = fmt.Sprintf("%s.%s", hostname, strings.TrimLeft(domain, "."))
		}
	}

	return
}

// createRuleCompDict parses a rule string and creates a map of each rule
// component's key mapped to its value. Duplicate keys are not allowed.
func createRuleCompDict(rule string) (map[string]string, error) {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return nil, ErrEmptyRule
	}

	// Split comma-separated values into rule components
	parts, err := parse.SplitCSV(rule)
	if err != nil {
		return nil, err
	}

	// Parse each rule component (key=val) and place into dictionary
	comps := make(map[string]string, len(parts))
	for idx, p := range parts {
		// Skip empty space
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// Split key=value
		key, val, ok := strings.Cut(p, "=")
		if !ok {
			return nil, NewErrKeyValFormat(idx, p)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" {
			return nil, NewErrNoKey(idx, p)
		}
		val, err := parse.Unquote(val)
		if err != nil {
			return nil, NewErrBadQuote(idx, p, err)
		}

		// Disallow duplicate keys
		if _, exists := comps[key]; exists {
			return nil, NewErrDuplicateKey(idx, p, key)
		}

		comps[key] = val
	}

	return comps, nil
}
