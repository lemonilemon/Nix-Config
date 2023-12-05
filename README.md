# nixos-config

This repository is maintained by lemonilemon. 

## Quick start

### In WSL (refers to [nixos-wsl-starter](https://github.com/LGUG2Z/nixos-wsl-starter))
- Get the latest [NixOS-WSL](https://github.com/nix-community/NixOS-WSL/releases/latest), then:

```sh
wsl --import NixOS .\NixOS\ nixos-wsl.tar.gz --version 2
```

- Install it in powershell or cmd using:

```sh
wsl --import NixOS .\NixOS\ .\nixos-wsl.tar.gz --version 2
```

- Enter your distro using the terminal you like

- Set up a channel using:

```sh
sudo nix-channel --add https://nixos.org/channels/nixos-23.05 nixos
sudo nix-channel --update
```

- Clone this repository: 

```sh
nix-shell -p git
git clone https://github.com/lemonilemon/nixos-config.git /tmp/configuration
```

- Apply the configuration by

```sh
sudo nixos-rebuild switch --flake /tmp/configuration#wsl
```

- Restart the WSL, then your configuration should be done

- Move the configuration to home directory:

```sh
mv /tmp/configuration ~/configuration
```

- Enjoy your WSL with NixOS!

---

## In development:

- Previewing Latex and markdown with neovim

- Get rid of vim-plug while using home-manager instead
