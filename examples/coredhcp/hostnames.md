<!--
SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC.
SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC

SPDX-License-Identifier: MIT
-->

# Hostname Rules for the CoreSMD CoreDHCP Plugin

## Contents

- [Hostname Rules for the CoreSMD CoreDHCP Plugin](#hostname-rules-for-the-coresmd-coredhcp-plugin)
  - [Contents](#contents)
  - [Quick Start](#quick-start)
  - [Summary](#summary)
  - [CoreSMD Configuration Directives](#coresmd-configuration-directives)
    - [`domain=DOMAIN`](#domaindomain)
    - [`hostname_log={info|debug|none}`](#hostname_loginfodebugnone)
    - [`hostname_rule=RULE`](#hostname_rulerule)
  - [Migrating from `*_pattern`](#migrating-from-_pattern)
  - [Pattern Syntax](#pattern-syntax)
    - [`{Nd}` - Zero-Padded NID](#nd---zero-padded-nid)
    - [`{id}` - Component Xname](#id---component-xname)
  - [Rule Syntax](#rule-syntax)
    - [Match Keys](#match-keys)
      - [`type:TYPE[|TYPE...]`](#typetypetype)
      - [`subnet:CIDR[|CIDR...]`](#subnetcidrcidr)
      - [`id:XNAME`](#idxname)
      - [`id_set:EXPR`](#id_setexpr)
    - [Action Keys](#action-keys)
      - [`pattern:PATTERN` (required)](#patternpattern-required)
      - [`domain:DOMAIN`](#domaindomain-1)
      - [`domain_append:{true|false}`](#domain_appendtruefalse)
      - [`continue:{true|false}`](#continuetruefalse)
      - [`name:STRING`](#namestring)
      - [`log:{info|debug|none}`](#loginfodebugnone)
  - [Rule Ordering](#rule-ordering)
  - [Examples](#examples)
    - [1. Minimal: NID hostnames for nodes only](#1-minimal-nid-hostnames-for-nodes-only)
    - [2. Add a domain suffix](#2-add-a-domain-suffix)
    - [3. Per-type rules](#3-per-type-rules)
    - [4. Subnet-specific BMC naming](#4-subnet-specific-bmc-naming)
    - [5. Suppress domain suffix for selected hosts](#5-suppress-domain-suffix-for-selected-hosts)
    - [6. Hostname override with `continue`](#6-hostname-override-with-continue)
    - [7. Familiar "legacy-style" configuration, expressed as rules](#7-familiar-legacy-style-configuration-expressed-as-rules)
  - [Caveats](#caveats)
  - [See Also](#see-also)

## Quick Start

In a CoreDHCP configuration:

```yaml
- coresmd: |
    svc_base_uri=https://smd.cluster.local
    ipxe_base_uri=http://192.168.1.1
    cache_valid=30s
    lease_time=24h
    domain=cluster.local
    hostname_log=info

    /* Nodes */
    hostname_rule=type:Node,pattern:nid{04d}

    /* BMCs */
    hostname_rule=type:NodeBMC,pattern:{id}
```

## Summary

CoreSMD can set DHCP option 12 (Host Name) using an ordered list of _hostname
rules_. Each rule matches on SMD inventory attributes (component type, component
ID, and the assigned IP address) and produces a hostname from a pattern.

Rules are evaluated in order. A matching rule produces a hostname and may stop
evaluation (the default or if `continue:false`) or allow evaluation to continue
(`continue:true`).

Within a `coresmd` configurationblock, comments use C-style delimiters: `/* ...
*/`.

## CoreSMD Configuration Directives

### `domain=DOMAIN`

The global domain suffix. When set, it is appended to hostnames unless a rule
specifies its own domain behavior. Leading dots are ignored.

**Default:** none

### `hostname_log={info|debug|none}`

Controls rule logging.

- `info`: log matching rules (rule name, match keys, resulting hostname)
- `debug`: log matching and non-matching rules (useful for troubleshooting)
- `none`: disable rule logging

**Default:** `none`

### `hostname_rule=RULE`

Add a hostname rule. May be specified multiple times. **Order matters!**

## Migrating from `*_pattern`

Older CoreSMD configurations used legacy pattern directives (for example
`node_pattern` and `bmc_pattern`). These are replaced by `hostname_rule=`.

The equivalent `hostname_rule=` forms are:

| **Old Config** | **New Config** |
|---|---|
| `node_pattern=PATTERN` | `hostname_rule=type:Node,pattern:PATTERN`    |
| `bmc_pattern=PATTERN`  | `hostname_rule=type:NodeBMC,pattern:PATTERN` |

When migrating, place these rules where you want them in the evaluation order.
It's common to put more specific rules before general ones so that the general
ones act as a catch-all.

## Pattern Syntax

### `{Nd}` - Zero-Padded NID

Generate a zero-padded Node ID where `N` is the number of digits:

| Pattern | NID | Result |
|---------|-----|--------|
| `{02d}` | 1   | `01`   |
| `{02d}` | 42  | `42`   |
| `{03d}` | 1   | `001`  |
| `{03d}` | 42  | `042`  |
| `{04d}` | 1   | `0001` |
| `{04d}` | 42  | `0042` |
| `{05d}` | 1   | `00001`|
| `{05d}` | 123 | `00123`|

### `{id}` - Component Xname

Use the full component identifier from SMD:

- Nodes: `x3000c0s0b0n0`, `x3000c0s1b0n1`, ...
- BMCs: `x3000c0s0b1`, `x3000c0s1b1`, ...

## Rule Syntax

`RULE` is a comma-separated list of `key:val` components organized by
"matching key-value pairs followed by "action" key-value pairs:

```
hostname_rule=<match_keyval(s)>,<action_keyval(s)>
```

A basic example of this is:

```
hostname_rule=type:Node,subnet:172.16.0.0/21,pattern:nid{04d}
```

(`type` and `subnet` are match key-value pairs, `pattern` is action.)

Values may be quoted with single or double quotes:

```
hostname_rule=type:Node,pattern:"rack42-{id}"
```

Unknown keys are rejected as an error. Duplicate keys in a rule are rejected as
an error.

If no rule matches, CoreSMD falls back to the built-in default pattern:
`unknown-{04d}`

### Match Keys

If a match key is omitted, it does not constrain the match. For example, if
`type` is omitted, any type (conforming to any further constraints) will match.

#### `type:TYPE[|TYPE...]`

Match component type as reported by SMD, e.g. `Node`, `NodeBMC`, `HSNSwitch`.
Tokens are whitespace-trimmed. If `type:` is present it must contain at least
one type.

**Default:** omitted (matches all types)

#### `subnet:CIDR[|CIDR...]`

Match assigned IP against one or more CIDRs. **Only the first IP is
matched** because CoreDHCP only assigns one address to the interface and it is
assumed that the first IP address in the list is the desired one.

**Default:** omitted (matches any subnet)

#### `id:XNAME`

Match the component's ID (xname).

Mutually exclusive with `id_set`.

**Default:** omitted (matches any ID)

#### `id_set:EXPR`

Match a set of component IDs.

**NOTE:** `id_set` is reserved; rules using it will fail to parse until the
ID-set matcher is implemented.

Mutually exclusive with `id`.

### Action Keys

#### `pattern:PATTERN` (required)

Hostname pattern to apply. See [Pattern Syntax](#pattern-syntax) above.

#### `domain:DOMAIN`

Domain to use with hostname if rule matches. Leading dots are ignored. See
`domain_append` below for modification of how domain behavior is applied.

If `domain:none`, no domain will be appended to the hostname, regardless of global `domain` or `domain_append` rule setting.

**Default:** omitted (global domain used, if set, otherwise none)

Default when omitted: use the global `domain` (if set)

#### `domain_append:{true|false}`

Controls whether the rule-specific `domain` is appended after the global
`domain` (`true`) or whether the rule-specific `domain` replaces the global
`domain` (`false`). This behavior only occurs when the rule-specific `domain`
is not set to `none`.

If the rule-specific `domain:none`, `domain_append` and the global `domain` have no effect. In other words, the hostname is not appended with any domain.

For instance:

- `domain_append:true`: the hostname is `<hostname>.<global_domain>.<rule_domain>`
- `domain_append:false`: the hostname is `<hostname>.<rule_domain>`
- `domain:none`: the hostname is `<hostname>`

**Default:** `false`

#### `continue:{true|false}`

Controls whether to continue evaluating subsequent rules if the rule matches.

If `true`, evaluation continues after a match.

If `false`, evaluation ceases after a match and the hostname is final.

**Default:** `false`

#### `name:STRING`

Rule identifier used in logs.

**Default:** a generated name based on hash of rule (e.g. rule-b176a842)

#### `log:{info|debug|none}`

Per-rule logging override.

Possible values:

- `info`: log only matches for this rule
- `debug`: log matches and non-matches for this rule
- `none`: suppress rule logging for this rule

**Default:** value of `hostname_log`

## Rule Ordering

Rules are evaluated in the order they appear.

- If a rule matches and `continue:false`, evaluation stops.
- If a rule matches and `continue:true`, evaluation continues.
- If multiple rules match, the hostname reflects the last matching rule that
  stops evaluation (or the last rule in the list).
- If no rule matches, CoreSMD falls back to compile-time default of
  `unknown-{04d}`.

When `continue:true`, the computed hostname _from the last matching rule_ is
kept and may be modified/overridden by later matching rules. This is useful for
generic naming followed by narrow overrides.

## Examples

### 1. Minimal: NID hostnames for nodes only

```yaml
- coresmd: |
    svc_base_uri=https://smd.cluster.local
    ipxe_base_uri=http://192.168.1.1
    hostname_rule=type:Node,pattern:nid{04d}
```

Assign node hostnames from NID. Other types fall back to the default pattern
(`unknown-{04d}`) unless additional rules are added.

| NID | Type | Hostname |
|-----|------|----------|
| 1   | Node | `nid0001` |
| 42  | Node | `nid0042` |

### 2. Add a domain suffix

```yaml
- coresmd: |
    domain=cluster.local
    hostname_rule=type:Node,pattern:nid{04d}
    hostname_rule=type:NodeBMC,pattern:bmc{04d}
```

Assign hostnames for nodes and BMCs and append a global domain.

| NID | Type | Hostname |
|-----|------|----------|
| 1   | Node    | `nid0001.cluster.local` |
| 1   | NodeBMC | `bmc0001.cluster.local` |

### 3. Per-type rules

```yaml
- coresmd: |
    domain=lab.local
    hostname_rule=type:Node,pattern:compute-{05d}
    hostname_rule=type:NodeBMC,pattern:ipmi-{05d}
    hostname_rule=type:HSNSwitch,pattern:{id}
```

Use separate naming schemes per component type.

| Input | Type | Hostname |
|-------|------|----------|
| NID=7 | Node | `compute-00007.lab.local` |
| NID=7 | NodeBMC | `ipmi-00007.lab.local` |
| ID=`x3000c0h0s0` | HSNSwitch | `x3000c0h0s0.lab.local` |

### 4. Subnet-specific BMC naming

```yaml
- coresmd: |
    domain=lab.local
    hostname_rule=type:NodeBMC,subnet:172.16.10.0/24,pattern:bmc10-{04d}
    hostname_rule=type:NodeBMC,subnet:172.16.11.0/24,pattern:bmc11-{04d}
    hostname_rule=type:NodeBMC,pattern:bmc-{04d}
```

Name BMCs differently depending on the assigned subnet, with a final fallback
rule for BMCs.

| IP (assigned) | Type | NID | Hostname |
|--------------:|------|-----|----------|
| 172.16.10.99  | NodeBMC | 7 | `bmc10-0007.lab.local` |
| 172.16.11.50  | NodeBMC | 7 | `bmc11-0007.lab.local` |
| 172.16.12.10  | NodeBMC | 7 | `bmc-0007.lab.local` |

### 5. Suppress domain suffix for selected hosts

Use `domain:none` in the rule action. This overrides global `domain` and
overrides `domain_append:true`.

```yaml
- coresmd: |
    domain=.cluster.local
    hostname_rule=type:Node,pattern:nid{04d}

    /* BMCs get *no* domain suffix */
    hostname_rule=type:NodeBMC,pattern:bmc{04d},domain:none,domain_append:true
```

Use `domain:none` to suppress suffixing, even if `domain_append:true` is set.

| NID | Type | Hostname |
|-----|------|----------|
| 1 | Node | `nid0001.cluster.local` |
| 1 | NodeBMC | `bmc0001` |

### 6. Hostname override with `continue`

`continue:true` allows later rules to refine/override hostnames.

```yaml
- coresmd: |
    domain=cluster.local

    /* Base node naming */
    hostname_rule=name:base,type:Node,pattern:nid{04d},continue:true

    /* Override domain for a subset of nodes (example) */
    hostname_rule=name:mgmt,type:Node,subnet:172.16.0.0/24,pattern:nid{04d},domain:mgmt.local
```

Two rules match nodes in `172.16.0.0/24`. The first sets the baseline hostname;
the second overrides the domain for that subnet.

| IP (assigned) | Type | NID | Hostname |
|--------------:|------|-----|----------|
| 172.16.0.10 | Node | 7 | `nid0007.mgmt.local` |
| 172.16.9.10 | Node | 7 | `nid0007.cluster.local` |

### 7. Familiar "legacy-style" configuration, expressed as rules

The old `node_pattern`, `bmc_pattern`, `hostname_by_type`, and
`hostname_default` directives are replaced by `hostname_rule`.

```yaml
- coresmd: |
    domain=cluster.local

    /* Equivalent to: node_pattern=nid{04d} */
    hostname_rule=type:Node,pattern:nid{04d}

    /* Equivalent to: bmc_pattern=bmc{04d} */
    hostname_rule=type:NodeBMC,pattern:bmc{04d}

    /* Equivalent to: hostname_by_type=HSNSwitch:{id} */
    hostname_rule=type:HSNSwitch,pattern:{id}

    /* Equivalent to: hostname_default=... (final fallback) */
    hostname_rule=pattern:unknown-{04d}
```

This produces the same administrator-facing naming scheme as the legacy knobs,
while preserving rule ordering and allowing additional matching criteria.

## Caveats

- `pattern` is required.
- `type:` is optional, but if present must include at least one type.
- Only the first assigned IP is used for subnet matching.
- `domain:none` suppresses all domain suffixing (even if `domain_append:true`).
- `id_set` is reserved; rules using it will fail to parse until implemented.

If you do not want the built-in fallback (`unknown-{04d}`) to apply, provide a
final "catch-all" rule that matches your desired behavior, e.g.:

```yaml
- coresmd: |
    ...
    hostname_rule=pattern:{id}
```

## See Also

CoreDHCP configuration documentation and SMD inventory documentation.
