package hub

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/lucaswilliameufrasio/borda/internal/assistant"
)

// HubView is what the hub is allowed to keep after the person approves an agent.
// It has no router password and no username.
type HubView struct {
	Linked   bool                `json:"linked"`
	Name     string              `json:"name"`
	Portrait *assistant.Portrait `json:"portrait,omitempty"`
	Events   *[]Event            `json:"events,omitempty"`
}

type storedLink struct {
	RouterID   string `json:"router_id"`
	ApprovedAt string `json:"approved_at"`
}

type linkFile struct {
	Links []storedLink `json:"links"`
}

func (store *Store) loadLinks() error {
	body, err := os.ReadFile(store.linksPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var state linkFile
	if err := json.Unmarshal(body, &state); err != nil {
		return err
	}
	if state.Links == nil {
		state.Links = []storedLink{}
	}
	for _, link := range state.Links {
		if !idPattern.MatchString(link.RouterID) || link.ApprovedAt == "" {
			return os.ErrInvalid
		}
	}
	store.links = state.Links
	return nil
}

// Link records the person's approval. Without approval it does not publish anything.
func (store *Store) Link(ctx context.Context, id string, confirmed bool) (HubView, error) {
	if !confirmed {
		return HubView{}, &ValidationError{Fields: []FieldError{{
			Field:   "confirmed",
			Message: "O agent só entra no hub com aprovação.",
		}}}
	}
	router, err := store.find(id)
	if err != nil {
		return HubView{}, err
	}
	store.mu.Lock()
	if !store.hasLinkLocked(id) {
		if store.links == nil {
			store.links = []storedLink{}
		}
		store.links = append(store.links, storedLink{
			RouterID:   id,
			ApprovedAt: store.clock().Format(time.RFC3339),
		})
		if err := store.saveLinksLocked(); err != nil {
			store.links = store.links[:len(store.links)-1]
			store.mu.Unlock()
			return HubView{}, err
		}
	}
	store.mu.Unlock()
	return store.Hub(ctx, router.ID)
}

// Hub returns the published view. Before approval the hub has only the name.
func (store *Store) Hub(ctx context.Context, id string) (HubView, error) {
	router, err := store.find(id)
	if err != nil {
		return HubView{}, err
	}
	if !store.isLinked(id) {
		return HubView{Linked: false, Name: router.Name}, nil
	}
	snapshot, err := store.Snapshot(ctx, id)
	if err != nil {
		return HubView{}, err
	}
	events, err := store.Events(id)
	if err != nil {
		return HubView{}, err
	}
	portrait := assistant.PortraitOf(visibleReading(snapshot))
	return HubView{
		Linked:   true,
		Name:     router.Name,
		Portrait: &portrait,
		Events:   &events,
	}, nil
}

// RefuseHubWrite rejects a write sent to the hub. Confirmation still belongs to the agent.
func (store *Store) RefuseHubWrite(id string) error {
	if _, err := store.find(id); err != nil {
		return err
	}
	return &ValidationError{Fields: []FieldError{{
		Field:   "action",
		Message: "A escrita fica no agent e pede confirmação lá.",
	}}}
}

func (store *Store) isLinked(id string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.hasLinkLocked(id)
}

func (store *Store) hasLinkLocked(id string) bool {
	for _, link := range store.links {
		if link.RouterID == id {
			return true
		}
	}
	return false
}

func (store *Store) saveLinksLocked() error {
	state := linkFile{Links: store.links}
	if state.Links == nil {
		state.Links = []storedLink{}
	}
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	temporary := store.linksPath + ".tmp"
	if err := os.WriteFile(temporary, body, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, store.linksPath)
}
