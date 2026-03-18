// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC.
// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cache

import (
	"testing"
	"time"

	"github.com/openchami/coresmd/internal/smdclient"
)

func TestNewCache(t *testing.T) {
	type args struct {
		duration string
		client   *smdclient.SmdClient
	}

	tests := []struct {
		name         string
		args         args
		wantErr      bool
		wantDuration time.Duration
	}{
		{
			name: "valid_duration_and_client",
			args: args{
				duration: "10m",
				client:   &smdclient.SmdClient{},
			},
			wantErr:      false,
			wantDuration: 10 * time.Minute,
		},
		{
			name: "zero_duration_and_valid_client_ok",
			args: args{
				duration: "0s",
				client:   &smdclient.SmdClient{},
			},
			wantErr:      false,
			wantDuration: 0,
		},
		{
			name: "invalid_duration_string_returns_error",
			args: args{
				duration: "not_a_duration",
				client:   &smdclient.SmdClient{},
			},
			wantErr: true,
		},
		{
			name: "nil_client_returns_error",
			args: args{
				duration: "5m",
				client:   nil,
			},
			wantErr: true,
		},
		{
			name: "invalid_duration_and_nil_client_still_error",
			args: args{
				duration: "completely_wrong",
				client:   nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewCache(nil, tt.args.duration, tt.args.client)

			if (err != nil) != tt.wantErr {
				t.Fatalf("NewCache() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				if cache != nil {
					t.Fatalf("NewCache() cache = %#v, want nil on error", cache)
				}
				return
			}

			if cache == nil {
				t.Fatalf("NewCache() returned nil cache, want non-nil")
			}

			if cache.Client != tt.args.client {
				t.Errorf("Client = %#v, want %#v", cache.Client, tt.args.client)
			}

			if cache.Duration != tt.wantDuration {
				t.Errorf("Duration = %v, want %v", cache.Duration, tt.wantDuration)
			}

			if !cache.LastUpdated.IsZero() {
				t.Errorf("LastUpdated = %v, want zero value", cache.LastUpdated)
			}

			if cache.EthernetInterfaces != nil {
				t.Errorf("EthernetInterfaces = %#v, want nil (zero value)", cache.EthernetInterfaces)
			}

			if cache.Components != nil {
				t.Errorf("Components = %#v, want nil (zero value)", cache.Components)
			}
		})
	}
}
