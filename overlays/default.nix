{ inputs, ... }:
{
  nixpkgs.overlays = [
    # AI coding agents — packages live under `pkgs.llm-agents.*`. The flake's
    # pinned nixpkgs is used on purpose so builds hit cache.numtide.com.
    inputs.llm-agents.overlays.default

    (final: prev: {
      openusage = prev.buildGo125Module rec {
        pname = "openusage";
        version = "0.22.0";

        src = prev.fetchFromGitHub {
          owner = "janekbaraniewski";
          repo = "openusage";
          rev = "v${version}";
          hash = "sha256-jB06xwK3egPoDIdvZq3LH5kAIRq8LAmZInzhCvEWJ24=";
        };

        vendorHash = "sha256-cBltKSILSp1tfie1dLoilu0jL/qDuHo2JUzMXb2DHno=";
        subPackages = [ "cmd/openusage" ];
        nativeBuildInputs = [ prev.go_1_25 ];
        env.CGO_ENABLED = "1";
        doCheck = false;

        meta = with prev.lib; {
          description = "Terminal-first local quota and usage tracking for AI coding tools";
          homepage = "https://openusage.sh/";
          license = licenses.mit;
          mainProgram = "openusage";
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
