package main

import "gsip/sip"

var deviceManager *Manager

func init() {
	deviceManager = &Manager{
		containers: sip.CreateSafeMap(1024),
	}
}

type Manager struct {
	containers *sip.SafeMap
}

func (d *Manager) Add(key string, device interface{}) {
	d.containers.Add(key, device)
}

func (d *Manager) Find(id string) interface{} {
	if find, b := d.containers.Find(id); b {
		return find
	}

	return nil
}

func (d *Manager) Remove(key string) {
	d.containers.Remove(key)
}
