---
name: nook-scene
description: Install the Nook menu-bar app, design a scene for it, and submit that scene to the catalog. Use when the user mentions Nook, a Nook scene, nook://scene, nook-scenes, or asks to install the app, make a scene, or publish one.
---

# Nook scenes

Nook is a macOS menu-bar app. Its flyout is one HTML scene, and the scene is the user's own control center. A scene is published only when its pull request merges into `sliiu/nook-scenes`.

The app lives at `sliiu/nook`. The catalog lives at `sliiu/nook-scenes`. Do not put a new scene in the app repository.

Do the three parts in order. Stop and ask before installing, before writing files, and before opening a pull request.

## Install the app

Check the machine before fetching anything.

```bash
uname -s
uname -m
```

Then list the assets of the latest release:

```bash
gh release view --repo sliiu/nook --json tagName,assets --jq '{tag:.tagName, assets:[.assets[].name]}'
```

Match the machine to an asset by its filename. `Darwin` goes with `.dmg`; `arm64` and `aarch64` go with `arm64`, and `x86_64` goes with `x64`. Anything else has no match today, and that is fine: match against the names that are actually there, not against a fixed list. A later release may add a new system or a new architecture, and the same rule covers it.

- One asset matches: install it. Only a `.dmg` has a known install. Mount it, copy the `.app` into `/Applications`, remove the quarantine flag, and open it.

```bash
gh release download --repo sliiu/nook --pattern '*.dmg' --dir "$TMPDIR"
hdiutil attach -nobrowse "$TMPDIR"/*.dmg
cp -R /Volumes/Nook*/Nook.app /Applications/
hdiutil detach /Volumes/Nook*
xattr -dr com.apple.quarantine /Applications/Nook.app
open /Applications/Nook.app
```

  The volume name and the app name come from the image, so check what mounted before copying. The app is not signed or notarized; the quarantine removal is what lets it open the first time. If the copy fails because the app is running, say so instead of forcing it.

- A matching asset is not a `.dmg`: stop. Say which system was detected and which files the release contains, and ask whether to skip the install and design the scene anyway. Do not invent install steps for a format that is not listed here.

- Nothing matches: stop the same way. Say which system and architecture were detected and which files the release contains, and ask whether to skip the install and design the scene anyway. A scene can be written and submitted from any system; only the preview needs the app.

## Design a scene

A scene is one self-contained HTML file, at most 1 MiB. Write it for the catalog sandbox, not for a file imported into the app. An imported file runs in the app's own document and can see everything the shell can see; a published scene runs in a sandboxed frame and gets the bridge below and nothing else. A scene that only works when imported will be refused.

Ask what the scene should do and look like before writing it. Keep it to one screen. The flyout has no room for a second view.

### The bridge

`Control` and `events` are globals. Every call returns a promise.

- `Control.getState()` returns `{ volume, brightness, brightnessAvailable }`. Read this once at start.
- `Control.listControls()` returns every control: `{ id, kind, available, adjustable, percent, muted, on, selected, options }`. `kind` is `level`, `switch`, `choice` or `action`. `options` is `{ id, label }`. The ids are `output.level`, `display.level`, `input.level`, `keyboard.level`, `output.device`, `media.previous`, `media.toggle`, `media.next`, `appearance` and `wifi.power`. An unavailable control stays in the list with `available: false`; show it, do not hide it or invent a value.
- `Control.setControl(id, write)` changes one control. `write` carries only the fields that change: `{ percent }`, `{ muted }`, `{ on }` or `{ selected }`. An action takes an empty write. A refused write rejects the promise.
- `Control.getVolume()` returns `{ level, muted, adjustable }`. `level` is 0–100.
- `Control.setVolume(level)` sets the output volume. A non-zero write unmutes.
- `Control.setMute(muted)` mutes or unmutes without changing the level.
- `Control.getBrightness()` returns `{ level, available }`. `available` is false when the machine has no built-in display, and `setBrightness` then does nothing. Keep the control on screen and show it as unavailable; do not hide the axis or invent a number.
- `Control.setBrightness(level)` sets the built-in display, 0–100.
- `Control.noteDrag(true)` while the pointer is down, `Control.noteDrag(false)` when it goes up. The flyout stays open during a drag.
- `Control.noteIdle()` once the pointer has been still. The short flyout hides itself about a second and a half later; the full one stays until the next click.
- `Control.setFlyoutSize(w, h)` asks for a size in pixels. Both sides must be from 1 to 1200. The shell already measures the scene's first element, so call this only to ask for a size the content does not already have.

Two events, each with `.on(fn)`, which returns an unsubscribe function:

- `events.stateChanged` fires when volume or brightness changes outside the page. The payload is `{ source, volume, brightness }`, and `source` is `"external"` for a change the page did not make. Ignore `source: "user"` or the page echoes its own writes. Either number may be absent.
- `events.pillForm` fires with `{ form }`, one of `"full"`, `"short"`, `"hidden"`.

Levels are integers from 0 to 100. Clamp before sending.

### What gets refused

The review rejects the page, and the app checks the same things again at install:

- `<script src>`, `<link href>`, and an `<img src>` that starts with `http:` or `https:`. A `data:` image is fine.
- `eval(`, `new Function`, `document.write`, `import(`, `fetch(`, `WebSocket(`, `XMLHttpRequest(`.
- `parent` or `top` used as a window: `parent.something`, `parent["something"]`, `window.parent`. A CSS property or a local name is not a use.

So no remote assets, no network, and no reaching out of the frame. Put the CSS and the script in the file. The scan only catches the obvious forms; the sandbox is what holds.

The shipped `home` scene is a useful reference for how the bridge feels. Read it for the interaction, and do not copy it. It is a large WebGL page, and a new scene should be its own thing.

### Preview

With the app installed, the user imports the file from Settings. Import only loads a preview; nothing is saved until they press Save. Ask them to look at it in the menu bar before submitting. Without the app, say that the preview was skipped and the page is untested.

## Submit the scene

One scene is one directory in `sliiu/nook-scenes`. The directory name is the id: lowercase letters, digits and hyphens, 2 to 40 characters, not starting or ending with a hyphen.

```text
night/
  scene.json
  scene.html
  preview.png      (optional)
```

`scene.json` has only these fields:

```json
{
  "id": "night",
  "name": "Night",
  "version": "1.0.0",
  "description": "One line about the scene",
  "htmlSha256": "lowercase hex sha-256 of the scene.html bytes",
  "preview": "preview.png",
  "author": "Your name"
}
```

`preview` and `author` are optional. `name` and `author` are at most 40 characters. `version` is three numbers, such as `1.0.0`. `description` is a string.

Hash the file's bytes, not a re-encoded string:

```bash
shasum -a 256 night/scene.html
```

Check it the way the catalog will. The checker reads the scene from the working tree of a `nook-scenes` checkout, so run it there:

```bash
node review/check.mjs night/scene.json night/scene.html
```

It exits 1 and prints one reason per line. Fix those before opening the pull request.

Fork `sliiu/nook-scenes`, add the directory on a branch, and open a pull request. The repository's review workflow runs the same checker on the pull request. Stop at the open pull request. Merging publishes the scene, and that is the maintainer's decision, not part of this skill.

### Updating

Change the same directory and open another pull request. The id stays, so the app replaces the installed scene instead of adding a second one. Raise the version: the last number for a small change, the middle for new behavior, the first when the way it works changes. Recompute `htmlSha256` whenever `scene.html` changes.

A `nook://scene/<id>` link names no commit, so it installs whatever `main` has. After the merge the old link points at the new version.
