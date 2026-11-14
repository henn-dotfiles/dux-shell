# DMS 2.1.0 (henn)

This is upstream [dux-shell](https://github.com/duxlabs-dev/dux-shell) (DMS)
v2.1.0 with local patches. Each patch lives on its own `henn/<name>` branch,
and `henn/main` merges all of them on top of the upstream release.

| Branch | Patch |
|---|---|
| `henn/osd-volume-at-limit` | Show the volume OSD when a volume key hits 0% or 100% |
| `henn/labwc-current-workspace` | Current-workspace window filter on labwc |

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
git checkout -B henn/main <new-tag>
git merge --no-ff henn/osd-volume-at-limit
git merge --no-ff henn/labwc-current-workspace
```

Then add this file back to `henn/main`.
