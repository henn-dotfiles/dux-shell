# DMS 2.1.0 (henn)

This is upstream [dux-shell](https://github.com/duxlabs-dev/dux-shell) (DMS)
v2.1.0 with local patches. Each patch lives on its own `henn/<name>` branch,
and `henn/main` merges all of them on top of the upstream release.

| Branch | Patch |
|---|---|
| `henn/osd-volume-at-limit` | Show the volume OSD when a volume key hits 0% or 100% |

## henn/osd-volume-at-limit

The volume OSD and the island only opened when the sink volume changed. At
100%, pressing volume up changes nothing, so no popup appeared. The patch adds
the `AudioService.sinkVolumeAtLimit` signal and opens the OSD and the island
on it.

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
git checkout -B henn/main <new-tag>
git merge --no-ff henn/osd-volume-at-limit
```

Then add this file back to `henn/main`.
