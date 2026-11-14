// Package extworkspace adds and removes workspaces through ext-workspace-v1.
//
// Quickshell 0.3.2 has no way to create an ext-workspace workspace, and its
// Windowset.canRemove is never set, so remove() always refuses. The shell
// asks the server instead, which sends the request on a short-lived Wayland
// connection.
package extworkspace

import (
	"fmt"

	"github.com/duxlabs-dev/dux-go/ipc"
	client "github.com/duxlabs-dev/dux-go/wayland/client"
	"github.com/duxlabs-dev/dux-shell/core/internal/proto/ext_workspace"
	"github.com/duxlabs-dev/dux-shell/core/internal/server/models"
	wlhelpers "github.com/duxlabs-dev/dux-shell/core/internal/wayland/client"
)

const (
	groupCapCreateWorkspace = 1
	workspaceCapRemove      = 4
)

type workspace struct {
	handle *ext_workspace.ExtWorkspaceHandleV1
	name   string
	caps   uint32
	gone   bool
}

type group struct {
	handle  *ext_workspace.ExtWorkspaceGroupHandleV1
	caps    uint32
	outputs []*client.Output
}

type session struct {
	display     *client.Display
	ctx         *client.Context
	manager     *ext_workspace.ExtWorkspaceManagerV1
	groups      []*group
	workspaces  []*workspace
	outputNames map[*client.Output]string
}

// connect opens a Wayland connection and collects the workspace groups.
// The caller closes s.ctx.
func connect() (*session, error) {
	display, err := client.Connect("")
	if err != nil {
		return nil, err
	}
	s := &session{display: display, ctx: display.Context(), outputNames: map[*client.Output]string{}}

	registry, err := display.GetRegistry()
	if err != nil {
		s.ctx.Close()
		return nil, err
	}
	registry.SetGlobalHandler(func(e client.RegistryGlobalEvent) {
		switch e.Interface {
		case "wl_output":
			if e.Version < 4 {
				return
			}
			out := client.NewOutput(s.ctx)
			out.SetNameHandler(func(ev client.OutputNameEvent) {
				s.outputNames[out] = ev.Name
			})
			_ = registry.Bind(e.Name, e.Interface, 4, out)
		case ext_workspace.ExtWorkspaceManagerV1InterfaceName:
			m := ext_workspace.NewExtWorkspaceManagerV1(s.ctx)
			m.SetWorkspaceGroupHandler(func(ev ext_workspace.ExtWorkspaceManagerV1WorkspaceGroupEvent) {
				g := &group{handle: ev.WorkspaceGroup}
				g.handle.SetCapabilitiesHandler(func(c ext_workspace.ExtWorkspaceGroupHandleV1CapabilitiesEvent) {
					g.caps = c.Capabilities
				})
				g.handle.SetOutputEnterHandler(func(o ext_workspace.ExtWorkspaceGroupHandleV1OutputEnterEvent) {
					g.outputs = append(g.outputs, o.Output)
				})
				s.groups = append(s.groups, g)
			})
			m.SetWorkspaceHandler(func(ev ext_workspace.ExtWorkspaceManagerV1WorkspaceEvent) {
				w := &workspace{handle: ev.Workspace}
				w.handle.SetNameHandler(func(n ext_workspace.ExtWorkspaceHandleV1NameEvent) {
					w.name = n.Name
				})
				w.handle.SetCapabilitiesHandler(func(c ext_workspace.ExtWorkspaceHandleV1CapabilitiesEvent) {
					w.caps = c.Capabilities
				})
				w.handle.SetRemovedHandler(func(ext_workspace.ExtWorkspaceHandleV1RemovedEvent) {
					w.gone = true
				})
				s.workspaces = append(s.workspaces, w)
			})
			if registry.Bind(e.Name, e.Interface, 1, m) == nil {
				s.manager = m
			}
		}
	})

	// Globals, then the manager's and outputs' initial state
	for range 3 {
		if err := wlhelpers.Roundtrip(display, s.ctx); err != nil {
			s.ctx.Close()
			return nil, err
		}
	}
	return s, nil
}

// Supported reports whether some workspace group allows creating workspaces.
func Supported() bool {
	s, err := connect()
	if err != nil {
		return false
	}
	defer s.ctx.Close()
	for _, g := range s.groups {
		if g.caps&groupCapCreateWorkspace != 0 {
			return true
		}
	}
	return false
}

// Create asks the compositor to add a workspace to the group shown on
// output (any group that allows it if output is empty or not found). An
// empty name lets the compositor pick one.
func Create(output, name string) error {
	s, err := connect()
	if err != nil {
		return err
	}
	defer s.ctx.Close()
	if s.manager == nil {
		return fmt.Errorf("compositor does not support ext-workspace-v1")
	}

	var target *group
	for _, g := range s.groups {
		if g.caps&groupCapCreateWorkspace == 0 {
			continue
		}
		if target == nil {
			target = g
		}
		for _, o := range g.outputs {
			if output != "" && s.outputNames[o] == output {
				target = g
			}
		}
	}
	if target == nil {
		return fmt.Errorf("compositor does not allow creating workspaces")
	}

	if err := target.handle.CreateWorkspace(name); err != nil {
		return err
	}
	if err := s.manager.Commit(); err != nil {
		return err
	}
	return wlhelpers.Roundtrip(s.display, s.ctx)
}

// Remove asks the compositor to remove the workspace called name.
func Remove(name string) error {
	s, err := connect()
	if err != nil {
		return err
	}
	defer s.ctx.Close()
	if s.manager == nil {
		return fmt.Errorf("compositor does not support ext-workspace-v1")
	}
	for _, w := range s.workspaces {
		if w.gone || w.name != name {
			continue
		}
		if w.caps&workspaceCapRemove == 0 {
			return fmt.Errorf("workspace %q cannot be removed", name)
		}
		if err := w.handle.Remove(); err != nil {
			return err
		}
		if err := s.manager.Commit(); err != nil {
			return err
		}
		return wlhelpers.Roundtrip(s.display, s.ctx)
	}
	return fmt.Errorf("no workspace %q", name)
}

func HandleRemove(conn *ipc.ConnWriter, req ipc.Request) {
	name, _ := req.Get[string]("name")
	if name == "" {
		conn.RespondError(req.ID, "missing workspace name")
		return
	}
	if err := Remove(name); err != nil {
		conn.RespondError(req.ID, err.Error())
		return
	}
	conn.Respond(req.ID, models.SuccessResult{Success: true})
}

func HandleSupported(conn *ipc.ConnWriter, req ipc.Request) {
	conn.Respond(req.ID, map[string]bool{"supported": Supported()})
}

func HandleCreate(conn *ipc.ConnWriter, req ipc.Request) {
	output, _ := req.Get[string]("output")
	name, _ := req.Get[string]("name")
	if err := Create(output, name); err != nil {
		conn.RespondError(req.ID, err.Error())
		return
	}
	conn.Respond(req.ID, models.SuccessResult{Success: true})
}
