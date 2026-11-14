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

Only the QML shell (`quickshell/`) is patched. It must match the version of
the `dms` binary from the PPA, so keep that package on the same version
(`apt-mark hold dms`).

```sh
git clone -b henn/main --recurse-submodules https://github.com/henn-dotfiles/dux-shell.git
sudo mkdir -p /opt/henn/share/dms
sudo cp -rL dux-shell/quickshell/. /opt/henn/share/dms/
```

Point the `dms` service at it with a systemd drop-in
(`~/.config/systemd/user/dms.service.d/shell-dir.conf`):

```ini
[Service]
Environment=DMS_SHELL_DIR=/opt/henn/share/dms
```

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
