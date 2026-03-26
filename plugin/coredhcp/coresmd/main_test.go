// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC.
// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package coresmd

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/openchami/coresmd/internal/cache"
	"github.com/openchami/coresmd/internal/tftp"
)

func TestConfigString_IncludesKeyFields(t *testing.T) {
	svc, _ := url.Parse("https://svc.example.test")
	ipxe, _ := url.Parse("https://ipxe.example.test")
	cacheDur := 10 * time.Second
	leaseDur := 5 * time.Minute

	cfg := Config{
		svcBaseURI:  svc,
		ipxeBaseURI: ipxe,
		caCert:      "/etc/ssl/ca.pem",
		cacheValid:  &cacheDur,
		leaseTime:   &leaseDur,
		singlePort:  true,
		tftpDir:     "/tftp",
		tftpPort:    1069,
		domain:      "example.test",
		hostnameLog: "debug",
	}

	s := cfg.String()
	wantSubstrings := []string{
		"svc_base_uri=" + svc.String(),
		"ipxe_base_uri=" + ipxe.String(),
		"ca_cert=/etc/ssl/ca.pem",
		"cache_valid=" + cacheDur.String(),
		"lease_time=" + leaseDur.String(),
		"single_port=true",
		"tftp_dir=/tftp",
		"tftp_port=1069",
		"domain=example.test",
	}
	for _, sub := range wantSubstrings {
		if !strings.Contains(s, sub) {
			t.Fatalf("Config.String() missing %q in %q", sub, s)
		}
	}
}

func TestParseConfig_HostnameRule(t *testing.T) {
	cacheDur := 15 * time.Second
	leaseDur := 30 * time.Minute

	args := []string{
		"svc_base_uri=https://svc.example.test",
		"ipxe_base_uri=https://ipxe.example.test",
		"ca_cert=/etc/pki/ca.pem",
		"cache_valid=" + cacheDur.String(),
		"lease_time=" + leaseDur.String(),
		"single_port=true",
		"tftp_dir=/tftp",
		"tftp_port=1069",
		"domain=cluster.local",
		"hostname_log=info",
		"hostname_rule=name:special,type:Node,id:x1000s0c0b0n0,pattern:login-{id},domain:mgmt.local",
	}

	cfg, errs := parseConfig(args...)
	if len(errs) != 0 {
		t.Fatalf("parseConfig() unexpected errors: %v", errs)
	}
	if cfg.hostnameLog != "info" {
		t.Fatalf("hostnameLog=%q", cfg.hostnameLog)
	}
	if len(cfg.hostnameRules) != 1 {
		t.Fatalf("hostnameRules len=%d, want 1", len(cfg.hostnameRules))
	}
	found := false
	for _, r := range cfg.hostnameRules {
		if r.Name == "special" {
			found = true
			if r.Match.ID != "x1000s0c0b0n0" {
				t.Fatalf("special rule id=%q", r.Match.ID)
			}
			if r.Match.Types == nil || !r.Match.Types["Node"] {
				t.Fatalf("special rule type match missing")
			}
			if r.Action.Pattern != "login-{id}" {
				t.Fatalf("special rule pattern=%q", r.Action.Pattern)
			}
			if r.Action.Domain != "mgmt.local" {
				t.Fatalf("special rule domain=%q", r.Action.Domain)
			}
		}
	}
	if !found {
		t.Fatalf("did not find explicit hostname_rule named 'special'")
	}
}

func TestConfigValidate_DefaultsApplied(t *testing.T) {
	svc, _ := url.Parse("https://svc.example.test")
	ipxe, _ := url.Parse("https://ipxe.example.test")

	cfg := Config{svcBaseURI: svc, ipxeBaseURI: ipxe}
	warns, errs := cfg.validate()
	if len(errs) != 0 {
		t.Fatalf("validate() errs=%v", errs)
	}
	if len(warns) == 0 {
		t.Fatalf("validate() expected warnings, got none")
	}
	if cfg.cacheValid == nil || cfg.cacheValid.String() != cache.DefaultCacheValid {
		t.Fatalf("cacheValid=%v want %s", cfg.cacheValid, cache.DefaultCacheValid)
	}
	if cfg.leaseTime == nil || cfg.leaseTime.String() != defaultLeaseTime {
		t.Fatalf("leaseTime=%v want %s", cfg.leaseTime, defaultLeaseTime)
	}
	if cfg.tftpPort != tftp.DefaultTFTPPort {
		t.Fatalf("tftpPort=%d want %d", cfg.tftpPort, tftp.DefaultTFTPPort)
	}
	if cfg.tftpDir != tftp.DefaultTFTPDirectory {
		t.Fatalf("tftpDir=%q want %q", cfg.tftpDir, tftp.DefaultTFTPDirectory)
	}
}

func TestSetup6_InvalidConfigFails(t *testing.T) {
	if Plugin.Setup6 == nil {
		t.Fatal("Plugin.Setup6 is nil")
	}
	h, err := Plugin.Setup6()
	if err == nil {
		t.Fatalf("setup6() with no args: expected error")
	}
	if h != nil {
		t.Fatalf("setup6() with invalid config: expected nil handler")
	}
}
