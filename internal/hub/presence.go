package hub

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"time"
)

var deviceIDPattern = regexp.MustCompile(`^[0-9a-f]{12}$`)

const (
	PresenceNow    = "now"
	PresenceRecent = "recent"
	PresenceAbsent = "absent"
)

const (
	presenceRecentWindow = time.Hour
	presenceRetention    = 7 * 24 * time.Hour
)

type storedPresence struct {
	RouterID string `json:"router_id"`
	ID       string `json:"id"`
	MAC      string `json:"mac"`
	Address  string `json:"address,omitempty"`
	Name     string `json:"name,omitempty"`
	SeenOn   string `json:"seen_on,omitempty"`
	LastSeen string `json:"last_seen"`
}

type presenceFile struct {
	Devices []storedPresence `json:"devices"`
}

func (store *Store) loadPresence() error {
	body, err := os.ReadFile(store.presencePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var state presenceFile
	if err := json.Unmarshal(body, &state); err != nil {
		return err
	}
	if state.Devices == nil {
		state.Devices = []storedPresence{}
	}
	for _, device := range state.Devices {
		if !idPattern.MatchString(device.RouterID) || !deviceIDPattern.MatchString(device.ID) {
			return os.ErrInvalid
		}
		if _, err := time.Parse(time.RFC3339, device.LastSeen); err != nil {
			return os.ErrInvalid
		}
	}
	store.presence = state.Devices
	return nil
}

// notePresence stores one row per device. A later sighting updates that row.
// It does not append a sample for every read. Rows older than 7 days are removed.
func (store *Store) notePresence(routerID string, current []DeviceJSON) ([]DeviceJSON, error) {
	now := store.clock()
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.presence == nil {
		store.presence = []storedPresence{}
	}
	seen := map[string]bool{}
	for _, device := range current {
		if !deviceIDPattern.MatchString(device.ID) {
			continue
		}
		seen[device.ID] = true
		store.upsertPresenceLocked(routerID, device, now)
	}
	store.dropExpiredLocked(routerID, now)
	if err := store.savePresenceLocked(); err != nil {
		return nil, err
	}
	return store.projectPresenceLocked(routerID, seen, now), nil
}

func (store *Store) upsertPresenceLocked(routerID string, device DeviceJSON, now time.Time) {
	for index := range store.presence {
		row := &store.presence[index]
		if row.RouterID != routerID || row.ID != device.ID {
			continue
		}
		row.MAC = device.MAC
		row.LastSeen = now.Format(time.RFC3339)
		if device.Address != "" {
			row.Address = device.Address
		}
		if device.Name != "" {
			row.Name = device.Name
		}
		if device.SeenOn != "" {
			row.SeenOn = device.SeenOn
		}
		return
	}
	store.presence = append(store.presence, storedPresence{
		RouterID: routerID,
		ID:       device.ID,
		MAC:      device.MAC,
		Address:  device.Address,
		Name:     device.Name,
		SeenOn:   device.SeenOn,
		LastSeen: now.Format(time.RFC3339),
	})
}

func (store *Store) dropExpiredLocked(routerID string, now time.Time) {
	kept := make([]storedPresence, 0, len(store.presence))
	for _, row := range store.presence {
		if row.RouterID == routerID && presenceAge(row.LastSeen, now) > presenceRetention {
			continue
		}
		kept = append(kept, row)
	}
	store.presence = kept
}

func (store *Store) projectPresenceLocked(routerID string, seen map[string]bool, now time.Time) []DeviceJSON {
	devices := make([]DeviceJSON, 0)
	for _, row := range store.presence {
		if row.RouterID != routerID {
			continue
		}
		age := presenceAge(row.LastSeen, now)
		presence := ""
		if seen[row.ID] {
			presence = PresenceNow
		} else if age <= presenceRecentWindow {
			presence = PresenceRecent
		} else if age <= presenceRetention {
			presence = PresenceAbsent
		}
		if presence == "" {
			continue
		}
		devices = append(devices, DeviceJSON{
			ID:       row.ID,
			MAC:      row.MAC,
			Address:  row.Address,
			Name:     row.Name,
			SeenOn:   row.SeenOn,
			Presence: presence,
		})
	}
	sort.Slice(devices, func(left int, right int) bool {
		return devices[left].MAC < devices[right].MAC
	})
	return devices
}

func presenceAge(lastSeen string, now time.Time) time.Duration {
	seenAt, err := time.Parse(time.RFC3339, lastSeen)
	if err != nil {
		return presenceRetention + time.Second
	}
	return now.Sub(seenAt)
}

func (store *Store) savePresenceLocked() error {
	devices := store.presence
	if devices == nil {
		devices = []storedPresence{}
	}
	body, err := json.MarshalIndent(presenceFile{Devices: devices}, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	temporary := store.presencePath + ".tmp"
	if err := os.WriteFile(temporary, body, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, store.presencePath)
}
