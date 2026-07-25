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
          in
          pkgs.runCommand "eww-backend-tests"
            {
              nativeBuildInputs = [ pkgs.python3 ];
              PYTHONDONTWRITEBYTECODE = "1";
            }
            ''
              mkdir -p modules/desktop/home/hyprland/eww
              cp -R ${./tests} tests
              cp -R ${./modules/desktop/home/hyprland/eww/scripts} \
                modules/desktop/home/hyprland/eww/scripts

              # unittest discovery skips packageless directories in silence, so
              # a new test dir without __init__.py would leave this check green
              # while running none of its tests. Only directories that actually
              # hold test files need to be packages; tests/nix holds eval
              # assertions, not Python.
              for dir in $(find tests -type d); do
                # find, not a glob: stdenv sets nullglob, which would collapse an
                # unmatched `ls dir/test_*.py` into a bare `ls` that always succeeds.
                has_tests=$(find "$dir" -maxdepth 1 -name 'test_*.py' -print -quit)
                if [ -n "$has_tests" ] && [ ! -f "$dir/__init__.py" ]; then
                  echo "test directory $dir has no __init__.py; unittest would skip it" >&2
                  exit 1
                fi
              done

              python3 -m unittest discover -s tests -t . -v
              touch $out
            '';
      });

      devShells = eachSystem (system: {
        default = nixpkgs.legacyPackages.${system}.mkShell {
          inherit (self.checks.${system}.pre-commit-check) shellHook;

          # Matches the eww-backend check, so running the tests by hand in this
          # shell cannot leave __pycache__ dirs behind.
          PYTHONDONTWRITEBYTECODE = "1";

          buildInputs = with nixpkgs.legacyPackages.${system}; [
            nixfmt
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
