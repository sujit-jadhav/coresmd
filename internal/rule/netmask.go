// SPDX-FileCopyrightText: Copyright 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rule

import (
	"encoding/binary"
	"net"
)

// Taken from:
// https://github.com/coredhcp/coredhcp/blob/a0841cb3038f63e3f93e813648cea8641a3bc5c0/plugins/netmask/plugin.go#L57-L62
func checkValidNetmask(netmask net.IPMask) bool {
	netmaskInt := binary.BigEndian.Uint32(netmask)
	x := ^netmaskInt
	y := x + 1
	return (y & x) == 0
}
