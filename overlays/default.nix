{
  nixpkgs.overlays = [
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

        postFixup =
          (oldAttrs.postFixup or "")
          + ''
            local spotify_executable="$out/bin/spotify" # ADJUST PATH AS NEEDED

            # Check if the executable exists before wrapping
            if [ -f "$spotify_executable" ]; then
              wrapProgram "$spotify_executable" \
                --add-flags "--enable-features=UseOzonePlatform" \
                --add-flags "--ozone-platform=wayland" \
                --add-flags "--enable-wayland-ime" \
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
        postFixup =
          (oldAttrs.postFixup or "")
          + ''
            echo "Wrapping 1Password binary with Wayland flags..."
            # IMPORTANT: Verify the actual path to the 1password binary relative to $out
            local op_executable="$out/bin/1password" # ADJUST PATH AS NEEDED

            if [ -f "$op_executable" ]; then
              wrapProgram "$op_executable" \
                --add-flags "--enable-features=UseOzonePlatform" \
                --add-flags "--ozone-platform=wayland" \
                --add-flags "--enable-wayland-ime" \
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
