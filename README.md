# ![Nook](.github/logo.png) Nook

A pixel room in your menu bar for output volume and built-in display brightness. The interface is an HTML scene. The page is at [sliiu.github.io/nook](https://sliiu.github.io/nook/). [中文](README.zh.md)

Drag across the flyout: horizontal is volume, vertical is brightness. When keys change either, the flyout pops up for about 1.6 seconds.

## Install

Download the latest `.dmg` from [Releases](https://github.com/sliiu/nook/releases) and drag Nook to Applications. It is not signed or notarized, so the first launch takes a right-click → Open.

> Tested only on **Apple Silicon (arm64) + macOS 26.6.2**. Other chips and macOS versions are unverified; open an issue if something breaks.

## Scenes

A scene is one HTML file using the `Control` and `events` globals. The flyout's size, corners and background all come from the scene.

- **Home** ships with the app. You can edit it but not delete it; Restore factory Home writes the built-in one back.
- **Import** only loads the preview. Nothing is saved until you press Save.
- **Market** scenes come from [`sliiu/nook-scenes`](https://github.com/sliiu/nook-scenes) and are published only once a pull request is merged. Install checks the hash and runs them in a sandbox that can change only volume and brightness. The entry link is `nook://scene/<id>`.

## Develop

Requires Go 1.27.3+. Building on macOS needs `CGO_ENABLED=1`.

```bash
go mod tidy
CGO_ENABLED=1 go build -o pixel-control-center .
CGO_ENABLED=1 go run github.com/egoist/mygo/cmd/mygo build   # writes the .app and .dmg under dist
```

## Trust

- Scenes you write or import from a file run with the app's own permissions. Only import files you trust.
- Brightness works on the built-in display only, through a private macOS API. A system update can break it, and without a built-in display the vertical axis is preview-only.

## Credit

The pixel room and lens are adapted from [slider-resident](https://ludion.ai/p/slider-resident) (MIT, @Lattice_Node). The built-in scene is original and does not reuse that file.

## Support

If Nook is useful to you, you can buy me a coffee:

<a href="https://www.buymeacoffee.com/pax_z" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me a Coffee" height="60" width="217"></a>
