# nixos-config

This repository is maintained by lemonilemon. 

## Quick start

### In WSL (refers to [nixos-wsl-starter](https://github.com/LGUG2Z/nixos-wsl-starter))
- Get the latest [NixOS-WSL](https://github.com/nix-community/NixOS-WSL/releases/latest), then:

- Install it in powershell or cmd using:

```sh
wsl --import NixOS .\NixOS\ .\nixos-wsl.tar.gz --version 2
```

- Enter your distro using the terminal you like

```sh
wsl -d NixOS
```

- Set up a channel using:

```sh
sudo nix-channel --add https://nixos.org/channels/nixos-23.11 nixos
sudo nix-channel --update
```

- Clone this repository and apply the configuration by: 

```sh
nix-shell -p git --run "git clone https://github.com/lemonilemon/nixos-config.git /tmp/configuration &&
sudo nixos-rebuild switch --flake /tmp/configuration#NixOS-wsl"
```

- Restart the WSL, then your configuration should be done

```sh
wsl -t NixOS
wsl -d NixOS
```

- Move the configuration to home directory:

```sh
mv /tmp/configuration ~/nixos-config
```

- Enjoy your WSL with NixOS!

---

## In development:

- Previewing Latex and markdown with neovim
