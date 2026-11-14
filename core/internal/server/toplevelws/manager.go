// Package toplevelws moves windows to workspaces through the henn labwc
// fork's zhenn_toplevel_workspace_manager_v1, for dragging windows onto
// workspace buttons.
//
// It keeps its own wlr-foreign-toplevel handles on the shared Wayland
// connection. A Quickshell Toplevel is matched to one of them by app id,
// title and position among windows with the same app id and title, since
// both lists are in creation order. Window drops reported by the
// compositor are forwarded to the shell, which answers with the workspace
// under the cursor.
package toplevelws

import (
	"fmt"
	"sync"
	"sync/atomic"

	wlclient "github.com/duxlabs-dev/dux-go/wayland/client"
	"github.com/duxlabs-dev/dux-shell/core/internal/log"
	tw "github.com/duxlabs-dev/dux-shell/core/internal/proto/toplevel_workspace"
)

// DropEvent is sent to subscribers when a window drag ends over a panel.
type DropEvent struct {
	ID    uint64 `json:"id"`
	X     int32  `json:"x"`
	Y     int32  `json:"y"`
	AppID string `json:"appId"`
	Title string `json:"title"`
}

type toplevel struct {
	handle *tw.ZwlrForeignToplevelHandleV1
	appID  string
	title  string
}

type Poster interface {
	Post(fn func())
}

type Manager struct {
	poster Poster

	mu        sync.Mutex
	toplevels []*toplevel
	drops     map[uint64]*toplevel
	manager   *tw.ZhennToplevelWorkspaceManagerV1
	available atomic.Bool
	nextDrop  uint64

	subscribers sync.Map // id -> chan DropEvent
}

func NewManager(display *wlclient.Display, poster Poster) (*Manager, error) {
	m := &Manager{poster: poster, drops: map[uint64]*toplevel{}}
	ctx := display.Context()

	registry, err := display.GetRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}

	registry.SetGlobalHandler(func(e wlclient.RegistryGlobalEvent) {
		switch e.Interface {
		case tw.ZwlrForeignToplevelManagerV1InterfaceName:
			ftm := tw.NewZwlrForeignToplevelManagerV1(ctx)
			ftm.SetToplevelHandler(func(ev tw.ZwlrForeignToplevelManagerV1ToplevelEvent) {
				m.addToplevel(ev.Toplevel)
			})
			if err := registry.Bind(e.Name, e.Interface, min(e.Version, 3), ftm); err != nil {
				log.Errorf("ToplevelWorkspace: bind %s: %v", e.Interface, err)
			}
		case tw.ZhennToplevelWorkspaceManagerV1InterfaceName:
			hm := tw.NewZhennToplevelWorkspaceManagerV1(ctx)
			hm.SetToplevelDroppedHandler(func(ev tw.ZhennToplevelWorkspaceManagerV1ToplevelDroppedEvent) {
				m.handleDrop(ev)
			})
			if err := registry.Bind(e.Name, e.Interface, 1, hm); err != nil {
				log.Errorf("ToplevelWorkspace: bind %s: %v", e.Interface, err)
				return
			}
			m.mu.Lock()
			m.manager = hm
			m.mu.Unlock()
			m.available.Store(true)
			log.Info("ToplevelWorkspace: compositor supports moving windows to workspaces")
		}
	})
	return m, nil
}

func (m *Manager) Available() bool { return m.available.Load() }

func (m *Manager) addToplevel(handle *tw.ZwlrForeignToplevelHandleV1) {
	t := &toplevel{handle: handle}
	handle.SetAppIdHandler(func(ev tw.ZwlrForeignToplevelHandleV1AppIdEvent) {
		m.mu.Lock()
		t.appID = ev.AppId
		m.mu.Unlock()
	})
	handle.SetTitleHandler(func(ev tw.ZwlrForeignToplevelHandleV1TitleEvent) {
		m.mu.Lock()
		t.title = ev.Title
		m.mu.Unlock()
	})
	handle.SetClosedHandler(func(tw.ZwlrForeignToplevelHandleV1ClosedEvent) {
		m.mu.Lock()
		for i, other := range m.toplevels {
			if other == t {
				m.toplevels = append(m.toplevels[:i], m.toplevels[i+1:]...)
				break
			}
		}
		for id, other := range m.drops {
			if other == t {
				delete(m.drops, id)
			}
		}
		m.mu.Unlock()
		_ = handle.Destroy()
	})
	m.mu.Lock()
	m.toplevels = append(m.toplevels, t)
	m.mu.Unlock()
}

func (m *Manager) handleDrop(ev tw.ZhennToplevelWorkspaceManagerV1ToplevelDroppedEvent) {
	m.mu.Lock()
	var t *toplevel
	for _, other := range m.toplevels {
		if other.handle == ev.Toplevel {
			t = other
			break
		}
	}
	if t == nil {
		m.mu.Unlock()
		return
	}
	m.nextDrop++
	id := m.nextDrop
	// Only the latest drop can still be answered
	clear(m.drops)
	m.drops[id] = t
	event := DropEvent{ID: id, X: ev.X, Y: ev.Y, AppID: t.appID, Title: t.title}
	m.mu.Unlock()

	m.subscribers.Range(func(_, v any) bool {
		select {
		case v.(chan DropEvent) <- event:
		default:
		}
		return true
	})
}

func (m *Manager) send(t *toplevel, workspace string) error {
	m.mu.Lock()
	hm := m.manager
	m.mu.Unlock()
	if hm == nil {
		return fmt.Errorf("compositor cannot move windows to workspaces")
	}
	m.poster.Post(func() {
		if err := hm.MoveToWorkspace(t.handle, workspace); err != nil {
			log.Warnf("ToplevelWorkspace: move_to_workspace: %v", err)
		}
	})
	return nil
}

// Move moves the index-th window (0-based, in creation order) with the
// given app id and title to the workspace.
func (m *Manager) Move(appID, title string, index int, workspace string) error {
	m.mu.Lock()
	var target *toplevel
	n := 0
	for _, t := range m.toplevels {
		if t.appID != appID || t.title != title {
			continue
		}
		if n == index {
			target = t
			break
		}
		n++
	}
	m.mu.Unlock()
	if target == nil {
		return fmt.Errorf("no window %q (%q) #%d", appID, title, index)
	}
	return m.send(target, workspace)
}

// MoveDropped moves the window of a drop event to the workspace.
func (m *Manager) MoveDropped(id uint64, workspace string) error {
	m.mu.Lock()
	t := m.drops[id]
	delete(m.drops, id)
	m.mu.Unlock()
	if t == nil {
		return fmt.Errorf("unknown or stale drop %d", id)
	}
	return m.send(t, workspace)
}

func (m *Manager) Subscribe(id string) chan DropEvent {
	ch := make(chan DropEvent, 16)
	m.subscribers.Store(id, ch)
	return ch
}

func (m *Manager) Unsubscribe(id string) {
	if v, ok := m.subscribers.LoadAndDelete(id); ok {
		close(v.(chan DropEvent))
	}
}
