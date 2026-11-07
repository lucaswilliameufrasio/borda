package hub

import (
	"encoding/json"
	"os"
	"time"
)

const (
	ReasonCheckFailed    = "check_failed"
	ReasonProofRequested = "proof_requested"
	ReasonRouteRestored  = "route_restored"
)

// Event is one path change stored by the hub.
type Event struct {
	ID     string `json:"id"`
	At     string `json:"at"`
	Reason string `json:"reason"`
}

type storedEvent struct {
	ID       string `json:"id"`
	RouterID string `json:"router_id"`
	At       string `json:"at"`
	Reason   string `json:"reason"`
}

type pathObservation struct {
	Gateway         string `json:"gateway"`
	InterfaceName   string `json:"interface_name,omitempty"`
	Distance        int    `json:"distance"`
	CheckResponding bool   `json:"check_responding"`
	Known           bool   `json:"known"`
}

type historyFile struct {
	Observations map[string]pathObservation `json:"observations"`
	Events       []storedEvent              `json:"events"`
}

func (store *Store) loadHistory() error {
	body, err := os.ReadFile(store.historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var state historyFile
	if err := json.Unmarshal(body, &state); err != nil {
		return err
	}
	if state.Observations == nil {
		state.Observations = map[string]pathObservation{}
	}
	if state.Events == nil {
		state.Events = []storedEvent{}
	}
	for _, event := range state.Events {
		if !idPattern.MatchString(event.ID) || !idPattern.MatchString(event.RouterID) || !knownReason(event.Reason) {
			return os.ErrInvalid
		}
	}
	store.observations = state.Observations
	store.events = state.Events
	return nil
}

func (store *Store) Events(id string) ([]Event, error) {
	if _, err := store.find(id); err != nil {
		return nil, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	events := make([]Event, 0)
	for index := len(store.events) - 1; index >= 0; index-- {
		event := store.events[index]
		if event.RouterID != id {
			continue
		}
		events = append(events, Event{ID: event.ID, At: event.At, Reason: event.Reason})
	}
	return events, nil
}

func (store *Store) notePath(routerID string, snapshot Snapshot) error {
	next := pathObservation{Known: true}
	if snapshot.ActiveRoute != nil {
		next.Gateway = snapshot.ActiveRoute.Gateway
		next.InterfaceName = snapshot.ActiveRoute.InterfaceName
		next.Distance = snapshot.ActiveRoute.Distance
	}
	if snapshot.Check != nil {
		next.CheckResponding = snapshot.Check.Responding
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.observations == nil {
		store.observations = map[string]pathObservation{}
	}
	previous, seen := store.observations[routerID]
	if seen && previous == next {
		return nil
	}
	changed := seen && pathChanged(previous, next)
	store.observations[routerID] = next
	if changed && snapshot.Check != nil && !snapshot.Check.Responding {
		if err := store.appendEventLocked(routerID, ReasonCheckFailed, store.clock()); err != nil {
			return err
		}
	}
	return store.saveHistoryLocked()
}

func (store *Store) noteProof(routerID string, started time.Time, restored bool) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.appendEventLocked(routerID, ReasonProofRequested, started); err != nil {
		return err
	}
	if !restored {
		return store.saveHistoryLocked()
	}
	if err := store.appendEventLocked(routerID, ReasonRouteRestored, store.clock()); err != nil {
		return err
	}
	return store.saveHistoryLocked()
}

func (store *Store) appendEventLocked(routerID string, reason string, at time.Time) error {
	if !knownReason(reason) {
		return os.ErrInvalid
	}
	id, err := newID()
	if err != nil {
		return err
	}
	if store.events == nil {
		store.events = []storedEvent{}
	}
	store.events = append(store.events, storedEvent{
		ID:       id,
		RouterID: routerID,
		At:       at.UTC().Format(time.RFC3339),
		Reason:   reason,
	})
	return nil
}

func (store *Store) saveHistoryLocked() error {
	state := historyFile{
		Observations: store.observations,
		Events:       store.events,
	}
	if state.Observations == nil {
		state.Observations = map[string]pathObservation{}
	}
	if state.Events == nil {
		state.Events = []storedEvent{}
	}
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	temporary := store.historyPath + ".tmp"
	if err := os.WriteFile(temporary, body, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, store.historyPath)
}

func (store *Store) clock() time.Time {
	if store.now != nil {
		return store.now().UTC()
	}
	return time.Now().UTC()
}

func pathChanged(previous pathObservation, next pathObservation) bool {
	if !previous.Known || !next.Known {
		return false
	}
	if previous.Gateway != next.Gateway {
		return true
	}
	if previous.InterfaceName != next.InterfaceName {
		return true
	}
	return previous.Distance != next.Distance
}

func knownReason(reason string) bool {
	if reason == ReasonCheckFailed {
		return true
	}
	if reason == ReasonProofRequested {
		return true
	}
	return reason == ReasonRouteRestored
}
