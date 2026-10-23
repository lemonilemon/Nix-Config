{
  description = "Lemonilemon's Nix Flake";

  nixConfig = {
    extra-substituters = [
      "https://cache.numtide.com"
      "https://lemonilemon.cachix.org"
      "https://aseipp-nix-cache.global.ssl.fastly.net"
      "https://mirrors.ustc.edu.cn/nix-channels/store"
    ];
    extra-trusted-public-keys = [
      "niks3.numtide.com-1:DTx8wZduET09hRmMtKdQDxNNthLQETkc/yaX7M4qK0g="
      "lemonilemon.cachix.org-1:3JBE3d5E5WuJRgOXNz+I5BUG+HRtBecADu0RBBJV1qI="
    ];
  };

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    nixos-hardware.url = "github:NixOS/nixos-hardware/master";
    nixos-wsl = {
      url = "github:nix-community/NixOS-WSL";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    nixvim = {
      url = "github:nix-community/nixvim";
    };
    home-manager = {
      url = "github:nix-community/home-manager";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    nix-index-database = {
      url = "github:nix-community/nix-index-database";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    sops-nix = {
      url = "github:Mic92/sops-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    systems.url = "github:nix-systems/default";
    flake-utils = {
      url = "github:numtide/flake-utils";
      inputs.systems.follows = "systems";
    };

    pre-commit-hooks.url = "github:cachix/git-hooks.nix";
    hyprland.url = "github:hyprwm/Hyprland";
    catppuccin.url = "github:catppuccin/nix";
    grub2-themes.url = "github:vinceliuice/grub2-themes";
    rose-pine-hyprcursor = {
      url = "github:ndom91/rose-pine-hyprcursor";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.hyprlang.follows = "hyprland/hyprlang";
    };
    sddm-orrery = {
      url = "github:lemonilemon/sddm-orrery";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    zen-browser = {
      url = "github:0xc000022070/zen-browser-flake";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    claude-desktop = {
      url = "github:k3d3/claude-desktop-linux-flake";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
    template-nvim = {
      url = "github:nvimdev/template.nvim/308f6f8f0bf98cb7c71855ffa8a3019a5642d1cd";
      flake = false;
    };
    coderunner-nvim = {
      url = "github:CRAG666/code_runner.nvim/45dfea066a6110abcbc3cd361457ac3cbaefd68b";
      flake = false;
    };
    copilot-lualine-nvim = {
      url = "github:AndreM222/copilot-lualine/222e90bd8dcdf16ca1efc4e784416afb5f011c31";
      flake = false;
    };
    llm-agents = {
      url = "github:numtide/llm-agents.nix";
    };
    # AI subscription usage tracker (Tauri app + headless CLI). Pinned to a
    # reviewed rev on purpose — its provider plugins read local CLI credentials,
    # so bumps should be deliberate (review the plugins/ diff, then move the pin).
    # The openusage-cli crate is not in any tagged release yet.
    openusage-community = {
      url = "github:openusage-community/openusage/eadcfe50dec5960dfff6afc0d32c8fa739c79376";
      flake = false;
    };
    claude-hud = {
      # v0.12.1
      url = "github:jarrodwatts/claude-hud/3e4e6b4a26e4a4263c6e0b8d60b5f124de4d84f0";
      flake = false;
    };
  };
  outputs =
    inputs@{
      self,
      systems,
      nixpkgs,
      ...
    }:
    let
      eachSystem = nixpkgs.lib.genAttrs (import systems);
      helpers = (import ./lib) { inherit inputs; };

      username = "lemonilemon";
      hostname = "SpaceNix";

    in
    {
      legacyPackages = eachSystem (
        system:
        (import ./nixpkgs {
          inherit inputs system;
        })
      );

      packages = eachSystem (system: {
        eww-bar = self.legacyPackages.${system}.callPackage ./packages/eww-bar/package.nix { };
      });

      homeManagerModules.eww-bar = ./packages/eww-bar/module.nix;

      formatter = eachSystem (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        pkgs.writeShellScriptBin "parallel-nixfmt" ''
          find . \( -name "*.nix" -not -path "./result*" -not -path "./.direnv/*" \) \
            -print0 | \
            ${pkgs.parallel}/bin/parallel --no-notice -0 -j$(nproc) ${pkgs.nixfmt}/bin/nixfmt
        ''
      );

      checks = eachSystem (system: {
        pre-commit-check = inputs.pre-commit-hooks.lib.${system}.run {
          src = ./.;
          hooks = {
            nixfmt = {
              enable = true;
              package = nixpkgs.legacyPackages.${system}.nixfmt;
            };
          };
        };

        host-options = import ./tests/host-options.nix {
          pkgs = nixpkgs.legacyPackages.${system};
          inherit (self) nixosConfigurations;
        };

        eww-bar-standalone =
          let
            pkgs = nixpkgs.legacyPackages.${system};
            home = inputs.home-manager.lib.homeManagerConfiguration {
              inherit pkgs;
              modules = [
                self.homeManagerModules.eww-bar
                {
                  home.username = "check";
                  home.homeDirectory = "/home/check";
                  home.stateVersion = "25.11";
                  programs.eww-bar.enable = true;
                }
              ];
            };
          in
          pkgs.writeText "eww-bar-standalone" (
            builtins.unsafeDiscardStringContext home.activationPackage.drvPath
          );

        eww-bar-self-contained = nixpkgs.legacyPackages.${system}.runCommand "eww-bar-self-contained" { } ''
          if grep -rn --include='*.nix' -F '../' ${./packages/eww-bar}; then
            echo "packages/eww-bar refers to files outside itself" >&2
            exit 1
          fi
          touch $out
        '';

        eww-backend =
          let
            pkgs = nixpkgs.legacyPackages.${system};
            checkYuckInitial = ./packages/eww-bar/tests/check_yuck_initial.py;
            checkGtkCss = ./packages/eww-bar/tests/check_gtk_css.py;
          in
          pkgs.runCommand "eww-backend-tests"
            {
              nativeBuildInputs = [
                pkgs.go
                pkgs.python3
              ];
            }
            ''
              # Not "go": that name collides with $GOPATH in the sandbox and the
              # toolchain then ignores the module's go.mod.
              cp -R ${./packages/eww-bar/go} src
              chmod -R u+w src

              # The sandbox has no HOME and no network; `go run` needs a
              # writable cache, and the module has no dependencies to fetch.
              export HOME=$TMPDIR
              export GOCACHE=$TMPDIR/go-cache
              export GOFLAGS=-mod=mod
              # net/http pulls cgo in for its resolver, and there is no reason
              # to link against libc for a test that resolves nothing.
              export CGO_ENABLED=0

              cd src
              go test ./...

              echo '{"fn":"DefaultSnapshot","args":[]}' \
                | go run ./internal/collect/diffgen > $TMPDIR/answer.json

              python3 ${checkYuckInitial} $TMPDIR/answer.json \
                ${./packages/eww-bar/config/eww.yuck}

              python3 ${checkGtkCss} \
                ${./packages/eww-bar/config/eww.scss}

              touch $out
            '';
      });

      devShells = eachSystem (system: {
        default = nixpkgs.legacyPackages.${system}.mkShell {
          inherit (self.checks.${system}.pre-commit-check) shellHook;

          buildInputs = with nixpkgs.legacyPackages.${system}; [
            nixfmt
            # The backend is a Go module now; `go test ./...` from this shell
            # is the same suite the eww-backend check runs.
            go
          ];
        };
      });

      # For NixOS
      nixosConfigurations = {
        NixOS-wsl = helpers.mkSystem {
          system = "x86_64-linux";
          inherit username hostname;
          profile = "wsl";
          isWSL = true;
        };

        laptop = helpers.mkSystem {
          system = "x86_64-linux";
          inherit username hostname;
          profile = "laptop";
        };

        desktop = helpers.mkSystem {
          system = "x86_64-linux";
          inherit username hostname;
          profile = "desktop";
        };
      };
    };
}
