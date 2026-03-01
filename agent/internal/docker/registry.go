package docker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

// CachedContainer holds static metadata for a Docker container.
// Populated once at agent start via ContainerList and updated by Docker events.
// This avoids calling ContainerList every second in the hot metrics path.
type CachedContainer struct {
	ID     string
	Name   string
	Image  string
	State  string
	Status string
	Labels map[string]string
}

// ContainerRegistry is a thread-safe in-memory cache of container metadata.
// It is the single source of truth for "which containers exist" during the
// hot metrics collection path. Only cgroup files are read per tick — the
// Docker API is only consulted at startup and on container lifecycle events.
type ContainerRegistry struct {
	mu         sync.RWMutex
	containers map[string]CachedContainer // keyed by full 64-char container ID
}

// NewContainerRegistry returns an empty, ready-to-use registry.
func NewContainerRegistry() *ContainerRegistry {
	return &ContainerRegistry{
		containers: make(map[string]CachedContainer),
	}
}

// Sync fetches the current container list from Docker and replaces the cache.
// Safe to call multiple times (e.g. after an event stream reconnect).
func (r *ContainerRegistry) Sync(ctx context.Context, cli *client.Client) error {
	list, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.containers = make(map[string]CachedContainer, len(list.Items))
	for _, c := range list.Items {
		name := c.Names[0]
		if len(name) > 0 && name[0] == '/' {
			name = name[1:]
		}
		r.containers[c.ID] = CachedContainer{
			ID:     c.ID,
			Name:   name,
			Image:  c.Image,
			State:  string(c.State),
			Status: string(c.Status),
			Labels: c.Labels,
		}
	}
	return nil
}

// List returns a point-in-time snapshot of all tracked containers.
func (r *ContainerRegistry) List() []CachedContainer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]CachedContainer, 0, len(r.containers))
	for _, c := range r.containers {
		result = append(result, c)
	}
	return result
}

func (r *ContainerRegistry) upsert(c CachedContainer) {
	r.mu.Lock()
	r.containers[c.ID] = c
	r.mu.Unlock()
}

func (r *ContainerRegistry) remove(id string) {
	r.mu.Lock()
	delete(r.containers, id)
	r.mu.Unlock()
}

func (r *ContainerRegistry) setState(id, state, status string) {
	r.mu.Lock()
	if c, ok := r.containers[id]; ok {
		c.State = state
		if status != "" {
			c.Status = status
		}
		r.containers[id] = c
	}
	r.mu.Unlock()
}

// WatchEvents keeps the registry in sync by consuming Docker's event stream.
// Should be launched as a dedicated goroutine — blocks until ctx is cancelled.
// Reconnects automatically and re-syncs after each stream error to avoid
// missing events during the gap.
func (r *ContainerRegistry) WatchEvents(ctx context.Context, cli *client.Client) {
	for {
		if ctx.Err() != nil {
			return
		}

		r.runEventLoop(ctx, cli)

		if ctx.Err() != nil {
			return
		}

		// Re-sync to catch any events missed while the stream was down.
		if err := r.Sync(ctx, cli); err != nil {
			log.Printf("[registry] re-sync after event stream error: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (r *ContainerRegistry) runEventLoop(ctx context.Context, cli *client.Client) {
	result := cli.Events(ctx, client.EventsListOptions{})
	eventsCh, errCh := result.Messages, result.Err

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			if err != nil {
				log.Printf("[registry] Docker events stream error: %v", err)
			}
			return
		case ev := <-eventsCh:
			if string(ev.Type) != "container" {
				continue
			}
			r.handleEvent(ctx, cli, ev)
		}
	}
}

func (r *ContainerRegistry) handleEvent(ctx context.Context, cli *client.Client, ev events.Message) {
	id := ev.Actor.ID
	if id == "" {
		return
	}

	switch string(ev.Action) {
	case "start":
		// "start" event fires before Docker updates the state in its API —
		// calling ContainerList here would often return state="created" instead
		// of "running". We know the container is running at this point, so set
		// it directly. If the container crashes immediately, the "die" event
		// will follow and correct the state to "exited".
		r.setState(id, "running", "")

	case "create":
		// "create" event fires when a new container is created but not yet started.
		// ContainerList is safe here — state is stable ("created").
		ctxT, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		list, err := cli.ContainerList(ctxT, client.ContainerListOptions{All: true})
		if err != nil {
			log.Printf("[registry] ContainerList after create event: %v", err)
			return
		}
		for _, c := range list.Items {
			if c.ID == id {
				name := c.Names[0]
				if len(name) > 0 && name[0] == '/' {
					name = name[1:]
				}
				r.upsert(CachedContainer{
					ID:     c.ID,
					Name:   name,
					Image:  c.Image,
					State:  string(c.State),
					Status: string(c.Status),
					Labels: c.Labels,
				})
				break
			}
		}

	case "die", "stop", "kill", "pause":
		r.setState(id, "exited", "")

	case "unpause":
		r.setState(id, "running", "")

	case "destroy":
		r.remove(id)

	case "rename":
		if newName, ok := ev.Actor.Attributes["name"]; ok {
			r.mu.Lock()
			if c, ok := r.containers[id]; ok {
				c.Name = newName
				r.containers[id] = c
			}
			r.mu.Unlock()
		}
	}
}
