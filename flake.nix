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

        host-options = import ./tests/nix/host-options.nix {
          pkgs = nixpkgs.legacyPackages.${system};
          inherit (self) nixosConfigurations;
        };

        eww-backend =
          let
            pkgs = nixpkgs.legacyPackages.${system};

            # This check runs the Go test suite AND the one assertion that
            # cannot live inside it: eww.yuck's :initial literal has to equal
            # the daemon's starting snapshot, and eww.yuck is not part of the Go
            # source.
            #
            # `go test` is here rather than left to buildGoModule's checkPhase
            # because `nix flake check` evaluates the NixOS configurations
            # without building them, so that checkPhase never ran in CI -- it
            # would first execute during a user's rebuild, which is the worst
            # place to discover a failing test. A timezone-dependent golden file
            # passed locally and failed in the sandbox for exactly that reason.
            #
            # It matters because that literal is what the bar renders for the
            # instant before the first line arrives on stdout. Drift shows up as
            # a flicker at startup or a missing widget, and nothing else in the
            # build would notice.
            #
            # A committed file rather than an inline string: Nix's
            # indented-string stripping and Python's indentation rules do not
            # agree, and a checker worth having is worth being able to read as
            # Python.
            checkYuckInitial = ./tests/nix/check_yuck_initial.py;

            # And the stylesheet's pseudo-classes, for the same reason: GTK
            # parses eww.scss, nothing in the build does, and GTK abandons the
            # rest of the file at the first rule it cannot read -- so one bad
            # selector strips the bar's appearance from that line down. It cost
            # a broken desktop once already.
            checkGtkCss = ./tests/nix/check_gtk_css.py;
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
              cp -R ${./modules/desktop/home/hyprland/eww/go} src
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
                ${./modules/desktop/home/hyprland/eww/eww.yuck}

              python3 ${checkGtkCss} \
                ${./modules/desktop/home/hyprland/eww/eww.scss}

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
