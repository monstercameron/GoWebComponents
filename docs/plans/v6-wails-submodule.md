# Wails source dependency

Added on 2026-09-08 for the V6 desktop integration.

| Field | Value |
| --- | --- |
| Repository | https://github.com/wailsapp/wails.git |
| Submodule path | `third_party/wails` |
| Release tag | `v3.0.0-beta.17` |
| Pinned commit | `5bce785eb1efbd121ef2e1cb2588eb50b9c59068` |
| Go module directory | `third_party/wails/v3` |
| Go import path | `github.com/wailsapp/wails/v3` |

The parent repository records an exact gitlink commit, not a moving branch.
The root GWC module does not require Wails. The isolated desktop example can
require the release version and use a local replacement to this checkout.
The Go module path is `/v3`, even though the upstream repository also contains
other versions and its website.

## Initial checkout

Run from the GWC repository root:

```powershell
git -c core.longpaths=true submodule update --init --depth 1 -- third_party/wails
git -C third_party/wails rev-parse HEAD
git submodule status -- third_party/wails
```

Wails includes generated test fixture paths longer than Windows' legacy path
limit. The command-scoped long-path option avoids a global Git setting. During
the initial local checkout, `core.longpaths=true` was set only in the Wails
submodule's local Git configuration. That local configuration is not versioned.

The existing Windows CI jobs that recursively initialize submodules (release,
doctor parity, and doclint commands) pass the same Git option through checkout-
scoped `GIT_CONFIG_COUNT`/key/value environment variables. This avoids depending
on runner-global Git defaults. Other workflows and the user's global Git config
are unchanged; the CI workflow changes still require a remote run for validation.

The initial add attempt incorrectly treated the release tag as a remote branch.
It was recovered in the newly created checkout by fetching the tag, checking
out its commit detached, and registering that checkout as the submodule. No
pre-existing project files were removed or reset.

## Updating the dependency

An upgrade is an intentional dependency change: fetch the selected release tag,
check out its commit in this submodule, align the example's required Go version
of Wails and any generated runtime/bindings, then run the desktop verification
workflow. Review both the gitlink and generated binding changes before recording
the upgrade in the parent repository. Do not use a moving branch or `@latest`
in the reproducible example build.

Upstream source is treated as read-only. If a fix is needed, document the exact
failure and keep any local patch explicit rather than silently altering the pin.
Retain upstream licensing and notices when packaging its code.

## Verification checkpoint

`git submodule status -- third_party/wails` resolves to the pinned commit above;
`git -C third_party/wails status --porcelain -uno` is empty after checkout.
Adding a Git submodule stages its `.gitmodules` entry and gitlink automatically.
They have not been committed or pushed by this integration effort.
