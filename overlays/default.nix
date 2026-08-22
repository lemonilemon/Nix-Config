{ inputs, ... }:
{
  nixpkgs.overlays = [
    # AI coding agents — packages live under `pkgs.llm-agents.*`. Use the
    # flake package set directly so store paths match cache.numtide.com.
    (final: _prev: {
      llm-agents = inputs.llm-agents.packages.${final.stdenv.hostPlatform.system};
    })

    # ccusage, taught the prices its own binary does not know.
    #
    # `--offline` is not a cache mode. It means "use the pricing table compiled
    # into the binary", and ccusage writes no pricing cache anywhere, so an
    # online run cannot warm a later offline one -- verified by running online
    # and watching a following offline run report the identical total. That
    # table is a snapshot frozen at the binary's release, and `claude-opus-5`
    # appears zero times in the 5 MB ELF, so every opus-5 day was priced at
    # $0.00: $616 of real spend over 45 days reported as free, silently, because
    # `--json` mode prints no missing-pricing warning at all.
    #
    # Dropping `--offline` is the obvious fix and is worse. It refetches ~3.2 MB
    # from GitHub and models.dev on EVERY invocation, and the eww bar runs this
    # on a 300 s poll -- 288 times a day, roughly 920 MB. It also turns a wrong
    # number into no number on a captive portal: a blackholed SYN retries three
    # times for 30.1 s measured, past the collector's 20 s ceiling, so the report
    # comes back empty and the bar goes stale.
    #
    # An override table costs nothing instead: 0.28 s against 0.31 s for plain
    # offline and ~3.2 s online, with no network at all. Verified against the
    # live table over two years of local history: 17 of the 18 models this host
    # has run match exactly, $3356.12 against $3356.21.
    #
    # The one exception is gemini-2.5-pro, and it is a limitation rather than a
    # typo. litellm prices it in two tiers -- input doubles and output rises by
    # half above 200k context -- and an override entry holds a single flat rate,
    # so long-context requests bill at the low tier. That is the whole $0.09
    # gap. Any model with a *_above_200k_tokens rate has the same ceiling.
    #
    # MAINTENANCE. These prices WIN over the live table, so a stale entry here
    # is confidently wrong where a missing one is merely zero. Re-check them
    # whenever the llm-agents input moves, and delete any the binary has since
    # learned. To re-verify:
    #
    #   diff <(ccusage daily --json --offline --since 20260701) \
    #        <(ccusage daily --json --since 20260701)
    #
    # The runtime backstop for the next model launch is in the Go collector,
    # which flags any model that burned tokens and priced at zero -- see
    # UnpricedModels in internal/collect/period.go.
    (final: prev: {
      llm-agents = prev.llm-agents // {
        ccusage = final.symlinkJoin {
          name = "ccusage-priced";
          paths = [ prev.llm-agents.ccusage ];
          nativeBuildInputs = [ final.makeWrapper ];
          postBuild = ''
            wrapProgram $out/bin/ccusage --add-flags "--config ${
              # Built from a Nix attrset rather than written as JSON text on
              # purpose: ccusage ignores a malformed config silently, exit 0 and
              # no stderr, which would revert every cost to $0 with nothing to
              # show for it. Generating the file makes a syntax error a Nix
              # evaluation failure instead.
              #
              # The nesting is load-bearing -- a top-level "pricingOverrides" is
              # accepted and then ignored; it has to sit under "defaults". Keys
              # match the model id EXACTLY, with no prefix or alias fallback, so
              # a dated id like claude-opus-5-20260601 would need its own entry.
              final.writeText "ccusage-pricing.json" (
                builtins.toJSON {
                  defaults.pricingOverrides = {
                    # Absent from the embedded table entirely.
                    "claude-opus-5" = {
                      inputCostPerToken = 5.0e-6;
                      outputCostPerToken = 2.5e-5;
                      cacheCreationInputTokenCost = 6.25e-6;
                      cacheReadInputTokenCost = 5.0e-7;
                    };
                    # Present but stale, and wrong in the other direction: the
                    # embedded price is 5x too HIGH, so these two were inflating
                    # the total while opus-5 deflated it.
                    "gpt-5.6-luna" = {
                      inputCostPerToken = 2.0e-7;
                      outputCostPerToken = 1.2e-6;
                      cacheCreationInputTokenCost = 2.5e-7;
                      cacheReadInputTokenCost = 2.0e-8;
                    };
                    "gpt-5.6-terra" = {
                      inputCostPerToken = 2.0e-6;
                      outputCostPerToken = 1.2e-5;
                      cacheCreationInputTokenCost = 2.5e-6;
                      cacheReadInputTokenCost = 2.0e-7;
                    };
                    # Gemini is absent from the embedded table wholesale, not
                    # just at the newest models -- every one of these read $0.00
                    # while the live table prices the same usage at $15.66.
                    # litellm lists no cache-creation cost for the family, so
                    # that key is omitted rather than guessed.
                    "gemini-2.5-flash" = {
                      inputCostPerToken = 3.0e-7;
                      outputCostPerToken = 2.5e-6;
                      cacheReadInputTokenCost = 3.0e-8;
                    };
                    "gemini-2.5-pro" = {
                      inputCostPerToken = 1.25e-6;
                      outputCostPerToken = 1.0e-5;
                      cacheReadInputTokenCost = 1.25e-7;
                    };
                    "gemini-3-flash-preview" = {
                      inputCostPerToken = 5.0e-7;
                      outputCostPerToken = 3.0e-6;
                      cacheReadInputTokenCost = 5.0e-8;
                    };
                    "gemini-3-pro-preview" = {
                      inputCostPerToken = 2.0e-6;
                      outputCostPerToken = 1.2e-5;
                      cacheReadInputTokenCost = 2.0e-7;
                    };
                    "gemini-3.1-pro-preview" = {
                      inputCostPerToken = 2.0e-6;
                      outputCostPerToken = 1.2e-5;
                      cacheReadInputTokenCost = 2.0e-7;
                    };
                  };
                }
              )
            }"
          '';
        };
      };
    })

    # OpenUsage Community's headless CLI (`openusage-cli probe` emits usage JSON).
    (final: prev: {
      openusage-cli = final.rustPlatform.buildRustPackage {
        pname = "openusage-cli";
        version = "0.6.37-unstable-2026-07-12";

        src = inputs.openusage-community;

        # The summary endpoint only says how many Codex manual resets exist.
        # Fetch their read-only detail endpoint too so the bar can show when
        # the earliest available reset expires. Drop after this lands upstream.
        patches = [ ./patches/openusage-codex-reset-credits.patch ];

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

    # libmamba 2.6.2 does not build against fmt 12.2: that release made
    # <fmt/core.h> include only <fmt/base.h>, so every `fmt::format` call
    # reached through it stops compiling. Upstream fixed the includes in mamba
    # 2.9.0, and nixpkgs has the bump queued in
    # https://github.com/NixOS/nixpkgs/pull/527994 — drop this once that lands.
    # mamba-cpp, micromamba and libmambapy follow along because they all
    # `inherit (libmamba) version src`.
    (final: prev: {
      libmamba = prev.libmamba.overrideAttrs (
        finalAttrs: _prevAttrs: {
          version = "2.9.0";
          src = final.fetchFromGitHub {
            owner = "mamba-org";
            repo = "mamba";
            tag = finalAttrs.version;
            hash = "sha256-Eh+EUI9TmnVjxgGp4yZhwzqa4WPr+fJzddIEqmAegIM=";
          };
        }
      );
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
