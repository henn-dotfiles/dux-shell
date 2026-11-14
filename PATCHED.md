# DMS 2.1.0 (henn)

This is upstream [dux-shell](https://github.com/duxlabs-dev/dux-shell) (DMS)
v2.1.0 with local patches. Each patch lives on its own `henn/<name>` branch,
and `henn/main` merges all of them on top of the upstream release.

| Branch | Patch |
|---|---|
| `henn/osd-volume-at-limit` | Show the volume OSD when a volume key hits 0% or 100% |
| `henn/labwc-current-workspace` | Current-workspace window filter on labwc |
| `henn/workspace-add-remove` | Add and delete workspaces from the workspace switcher |
| `henn/workspace-window-drop` | Drop app icons or windows on a workspace button to move them there |

## henn/osd-volume-at-limit

The volume OSD and the island only opened when the sink volume changed. At
100%, pressing volume up changes nothing, so no popup appeared. The patch adds
the `AudioService.sinkVolumeAtLimit` signal and opens the OSD and the island
on it.

## henn/labwc-current-workspace

On labwc, `CompositorService.filterCurrentWorkspace` returned every window,
so the "Current workspace" option of the running apps widget did nothing.
The henn labwc fork (`henn/foreign-toplevel-workspace`) only reports
foreign-toplevel outputs for windows on a shown workspace. The patch keeps
the windows that are on the bar's screen. With stock labwc it acts like the
"Current display" filter.

## henn/workspace-add-remove

The workspace switcher ends with a "+" pill, drawn like an inactive
workspace, that adds a workspace on the bar's output. Right-clicking a
workspace opens a menu with "Delete". It shows only when an ext-workspace
group allows creating workspaces, which the henn labwc fork
(`henn/dynamic-workspaces`) does; labwc moves the deleted workspace's
windows to the current one.

Quickshell 0.3.2 can't create workspaces and never sets
`Windowset.canRemove`, so `remove()` always refuses. Both requests go through
the server (`extworkspace.supported`, `extworkspace.create`,
`extworkspace.remove`), which sends them on a short-lived Wayland connection.
`core/internal/proto/ext_workspace` holds the generated ext-workspace-v1
bindings.

## henn/workspace-window-drop

Stacked on `henn/workspace-add-remove`. Needs the henn labwc fork
(`henn/toplevel-workspace-drop`), whose private
`zhenn_toplevel_workspace_manager_v1` moves a wlr-foreign-toplevel handle to
a named workspace and reports where a window move ended over a panel.

- Running Apps icons can be dragged onto a workspace button; the window (or
  every window of a grouped app) moves there.
- A window dragged by its titlebar and released on a workspace button moves
  there, back at its geometry from before the drag. The switcher converts
  the compositor position to Qt coordinates, which differ when Qt applies
  its own scale (e.g. `QT_FONT_DPI`).

The server (`core/internal/server/toplevelws`) keeps its own foreign-toplevel
handles on the shared Wayland connection and matches a Quickshell toplevel
by app id, title and creation order. Shell side:
`Services/ToplevelWorkspaceService.qml`.

## Install

The `dms` binary embeds the QML shell (`quickshell/`), so build the binary
from this branch; the shell then always matches it:

```sh
git clone -b henn/main --recurse-submodules https://github.com/henn-dotfiles/dux-shell.git
cd dux-shell
GOTOOLCHAIN=auto make -C core dist ARCH=amd64 DIST_OSES=linux
sudo install -Dm755 core/bin/dms-linux-amd64 /opt/henn/bin/dms
```

[henn-dotfiles/desktop](https://github.com/henn-dotfiles/desktop) does this
and diverts `/usr/bin/dms` to the built binary.

## Updating

To move to a newer DMS release, rebase each patch branch onto the new tag,
then rebuild `henn/main` from the tag and merge the patch branches again:

```sh
git fetch https://github.com/duxlabs-dev/dux-shell.git --tags
git rebase --onto <new-tag> v2.1.0 henn/osd-volume-at-limit
git rebase --onto <new-tag> v2.1.0 henn/labwc-current-workspace
git rebase --onto <new-tag> v2.1.0 henn/workspace-add-remove
# stacked: <old-add-remove> is henn/workspace-add-remove before the rebase above
git rebase --onto henn/workspace-add-remove <old-add-remove> henn/workspace-window-drop
git checkout -B henn/main <new-tag>
git merge --no-ff henn/osd-volume-at-limit
git merge --no-ff henn/labwc-current-workspace
git merge --no-ff henn/workspace-add-remove
git merge --no-ff henn/workspace-window-drop
```

Then add this file back to `henn/main`.
