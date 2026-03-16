// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC.
// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/openchami/coresmd/internal/smdclient"
)

const DefaultCacheValid = "30s"

type Cache struct {
	Client      *smdclient.SmdClient
	Duration    time.Duration
	LastUpdated time.Time
	Mutex       sync.RWMutex
	Log         *logrus.Entry

	EthernetInterfaces map[string]smdclient.EthernetInterface
	Components         map[string]smdclient.Component
}

func NewCache(log *logrus.Entry, duration string, client *smdclient.SmdClient) (*Cache, error) {
	cacheDuration, err := time.ParseDuration(duration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cache duration: %w", err)
	}
	if client == nil {
		return nil, errors.New("new Client needs non-nil SmdClient")
	}
	if log == nil {
		log = logrus.NewEntry(logrus.New())
	}

	c := &Cache{
		Client:   client,
		Duration: cacheDuration,
		Log:      log,
	}

	return c, nil
}

func (c *Cache) Refresh() error {
	c.Log.Info("initiating cache refresh")

	if c == nil {
		return fmt.Errorf("cache is nil")
	}

	// Fetch data
	c.Log.Debug("fetching EthernetInterfaces")
	ethIfaceData, err := c.Client.APIGet("/hsm/v2/Inventory/EthernetInterfaces")
	if err != nil {
		return fmt.Errorf("failed to fetch EthernetInterfaces from SMD: %w", err)
	}
	c.Log.Debug("EthernetInterfaces: " + string(ethIfaceData))
	c.Log.Debug("fetching Components")
	compsData, err := c.Client.APIGet("/hsm/v2/State/Components")
	if err != nil {
		return fmt.Errorf("failed to fetch Components from SMD: %w", err)
	}
	c.Log.Debug("Components: " + string(compsData))

	// Unmarshal it
	c.Log.Debug("unmarshaling EthernetInterfaces")
	var ethIfaceSlice []smdclient.EthernetInterface
	err = json.Unmarshal(ethIfaceData, &ethIfaceSlice)
	if err != nil {
		return fmt.Errorf("failed to unmarshal EthernetInterface data: %w", err)
	}
	c.Log.Debug("unmarshaling Components")
	var compsStruct struct {
		Components []smdclient.Component `json:"Components"`
	}
	err = json.Unmarshal(compsData, &compsStruct)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Components data: %w", err)
	}

	// Organize it to be referenced via map
	c.Log.Debug("organizing EthernetInterfaces into map")
	eiMap := make(map[string]smdclient.EthernetInterface)
	for _, ei := range ethIfaceSlice {
		eiMap[ei.MACAddress] = ei
	}
	c.Log.Debug("organizing Component into map")
	compMap := make(map[string]smdclient.Component)
	for _, comp := range compsStruct.Components {
		compMap[comp.ID] = comp
	}

	// Update cache with info
	c.Log.Debug("updating cache with map data")
	c.Mutex.Lock()
	c.EthernetInterfaces = eiMap
	c.Components = compMap
	c.LastUpdated = time.Now()
	c.Mutex.Unlock()
	c.Log.Infof("Cache updated with %d EthernetInterfaces and %d Components", len(eiMap), len(compMap))
	c.Log.Debugf("EthernetInterfaces: %v", eiMap)
	c.Log.Debugf("Components: %v", compMap)

	return nil
}

func (c *Cache) RefreshLoop() {
	c.Log.Info("initiating cache refresh loop")
	c.Log.Infof("refreshing cache every duration: %s", c.Duration.String())

	// Initial refresh
	err := c.Refresh()
	if err != nil {
		c.Log.Errorf("failed to refresh cache: %v", err)
	}

	// ...then each duration
	ticker := time.NewTicker(c.Duration)
	go func() {
		for range ticker.C {
			err := c.Refresh()
			if err != nil {
				c.Log.Errorf("failed to refresh cache: %v", err)
			}
		}
	}()
}
