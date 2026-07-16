{ inputs, ... }:
{
  nixpkgs.overlays = [
    # AI coding agents — packages live under `pkgs.llm-agents.*`. The flake's
    # pinned nixpkgs is used on purpose so builds hit cache.numtide.com.
    inputs.llm-agents.overlays.shared-nixpkgs

    # OpenUsage Community's headless CLI (`openusage-cli probe` emits usage JSON).
    (final: prev: {
      openusage-cli = final.rustPlatform.buildRustPackage {
        pname = "openusage-cli";
        version = "0.6.37-unstable-2026-07-12";

        src = inputs.openusage-community;

        cargoLock = {
          lockFile = "${inputs.openusage-community}/Cargo.lock";
          outputHashes = {
            "tauri-nspanel-2.1.0" = "sha256-oqkQCTe4ohZYoTEPAJfsGC0RGVwaHrEik9iCOKbxPh0=";
            "tauri-plugin-aptabase-1.0.0" = "sha256-j5nbJkSkn/ZE1UYNobUIzz38kdvgcILHrfcERMD16uA=";
          };
        };

        cargoBuildFlags = [
          "-p"
          "openusage-cli"
        ];

        nativeBuildInputs = [
          final.pkg-config
          final.rustPlatform.bindgenHook
        ];

        buildInputs = [
          final.openssl
          final.dbus
        ];

        # Workspace tests cover the Tauri app too; the CLI is what we ship.
        doCheck = false;

        meta = with prev.lib; {
          description = "Headless CLI for OpenUsage Community's AI usage tracker";
          homepage = "https://github.com/openusage-community/openusage";
          license = licenses.mit;
          mainProgram = "openusage-cli";
        };
      };
    })

    # Use no-remote as default (firefox)
    (final: prev: {
      firefox = prev.firefox.overrideAttrs (oldAttrs: {
        buildCommand = ''
          ${oldAttrs.buildCommand}
          mv $out/bin/firefox $out/bin/firefox-bin
          cat > $out/bin/firefox <<EOF
          #!/bin/sh
          exec $out/bin/firefox-bin -no-remote "\$@"
          EOF
          chmod +x $out/bin/firefox
        '';
      });
    })

    (final: prev: {
      # --- Spotify Override ---
      # Add flags to the .desktop file for Wayland IME support
      spotify = prev.spotify.overrideAttrs (oldAttrs: {
        nativeBuildInputs = (oldAttrs.nativeBuildInputs or [ ]) ++ [ final.patchutils ];

        postFixup = (oldAttrs.postFixup or "") + ''
          local spotify_executable="$out/bin/spotify" # ADJUST PATH AS NEEDED

          # Check if the executable exists before wrapping
          if [ -f "$spotify_executable" ]; then
            wrapProgram "$spotify_executable" \
              --add-flags "--enable-wayland-ime" \
              --add-flags "--enable-features=WaylandLinuxDrmSyncobj" \
              # Optionally unset conflicting env vars if needed:
              # --unset ELECTRON_OZONE_PLATFORM_HINT \
              # --unset NIXOS_OZONE_WL
            echo "Finished wrapping Spotify."
          else
            echo "Warning: Spotify executable not found at $spotify_executable, skipping wrap." >&2
          fi
        '';
      }); # End spotify override

      # --- 1Password Override ---
      _1password-gui-beta = prev._1password-gui-beta.overrideAttrs (oldAttrs: {
        nativeBuildInputs = (oldAttrs.nativeBuildInputs or [ ]) ++ [ final.patchutils ];
        postFixup = (oldAttrs.postFixup or "") + ''
          echo "Wrapping 1Password binary with Wayland flags..."
          # IMPORTANT: Verify the actual path to the 1password binary relative to $out
          local op_executable="$out/bin/1password" # ADJUST PATH AS NEEDED

          if [ -f "$op_executable" ]; then
            wrapProgram "$op_executable" \
              --add-flags "--enable-wayland-ime" \
              --add-flags "--enable-features=WaylandLinuxDrmSyncobj" \
              # --unset ELECTRON_OZONE_PLATFORM_HINT \
              # --unset NIXOS_OZONE_WL
            echo "Finished wrapping 1Password."
          else
            echo "Warning: 1Password executable not found at $op_executable, skipping wrap." >&2
          fi
        '';
      }); # End onepassword-desktop override
    })
  ];
}
