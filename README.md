<div align="center">
  <h1>catnip-gio</h1>
  <img src=".github/screenshot01.png" width="400" />
  <p>

  GUI frontend in [Gio](https://gioui.org) for the
  [catnip](https://github.com/noriah/catnip) visualizer.

  </p>
</div>

## Build

It is recommended that you use the `nix-shell` to obtain the needed
dependencies. Otherwise, refer to Gio's instructions.

After that, build this like a regular Go program:

```sh
go build
```

## Usage

First, list all devices:

```sh
―❤―▶ ./catnip-gio -l
pipewire:
  - alsa_output.pci-0000_03_00.1.hdmi-stereo-extra4
  - alsa_output.pci-0000_0d_00.6.analog-stereo
  - easyeffects_sink
  - Firefox
  - io.github.celluloid_player.Celluloid
```

Then, pick the device:

```sh
―❤―▶ ./catnip-gio -b pipewire -d easyeffects_sink
```

For more configurations, see `-h`.
