{
  inputs,
  pkgs,
  config,
  lib,
  ...
}:
{
  imports = [ ./thunderbird.nix ];

  config = lib.mkIf config.home.gui.apps.enable {
    home.packages =
      with pkgs;
      [
        webcord
        spotify
        vlc
        tradingview
        gopeed
        podman-desktop
        pdfcraft
      ]
      # Gated on the vault flag rather than listed unconditionally, so one option
      # governs the app, nvim's workspace and the Syncthing folder together.
      ++ lib.optional config.home.general.obsidian.enable pkgs.obsidian
      ++ lib.optional config.home.gui.apps.deckcraft.enable pkgs.deckcraft
      ++ lib.optional config.home.gui.apps.gridcraft.enable pkgs.gridcraft
      ++ lib.optional config.home.gui.apps.wordcraft.enable pkgs.wordcraft
      ++ lib.optional config.home.gui.apps.aerion.enable (pkgs.aerion.override { withOAuth = true; });

    xdg.mimeApps = lib.mkMerge [
      (lib.mkIf config.home.gui.apps.deckcraft.enable {
        enable = true;
        defaultApplications = {
          "application/vnd.openxmlformats-officedocument.presentationml.presentation" = [
            "ai.storyteller.deckcraft.desktop"
          ];
          "application/vnd.openxmlformats-officedocument.presentationml.slideshow" = [
            "ai.storyteller.deckcraft.desktop"
          ];
        };
      })
      (lib.mkIf config.home.gui.apps.gridcraft.enable {
        enable = true;
        defaultApplications = {
          "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" = [
            "ai.storyteller.gridcraft.desktop"
          ];
          "application/vnd.ms-excel.sheet.macroEnabled.12" = [ "ai.storyteller.gridcraft.desktop" ];
          "application/vnd.oasis.opendocument.spreadsheet" = [ "ai.storyteller.gridcraft.desktop" ];
        };
      })
      (lib.mkIf config.home.gui.apps.wordcraft.enable {
        enable = true;
        defaultApplications = {
          "application/vnd.openxmlformats-officedocument.wordprocessingml.document" = [
            "ai.storyteller.wordcraft.desktop"
          ];
          "application/msword" = [ "ai.storyteller.wordcraft.desktop" ];
          "application/vnd.ms-word.document.macroEnabled.12" = [ "ai.storyteller.wordcraft.desktop" ];
          "application/vnd.oasis.opendocument.text" = [ "ai.storyteller.wordcraft.desktop" ];
          "application/rtf" = [ "ai.storyteller.wordcraft.desktop" ];
        };
      })
    ];

    programs.ssh = {
      enable = true;
      enableDefaultConfig = false;
      includes = [
        "~/.ssh/1Password/config"
        # Host inventory (Host aliases, IPs), decrypted by sops-nix from
        # secrets/ssh-hosts.conf. Edit with `sops secrets/ssh-hosts.conf`,
        # then rebuild. On a machine without the age key the file is simply
        # absent and OpenSSH ignores the Include.
        "~/.ssh/config.inventory"
        # Hand-edited scratch hosts (unversioned). For entries that churn too
        # often to deserve a sops edit + rebuild, or that should stay off git
        # entirely; OpenSSH ignores this too while the file does not exist.
        "~/.ssh/config.local"
      ];
      settings = {
        "*" = {
          ForwardAgent = true;
          IdentityAgent = "~/.1password/agent.sock";
        };
      };
    };

    sops = {
      age.keyFile = "${config.home.homeDirectory}/.config/sops/age/keys.txt";
      secrets."ssh-hosts" = {
        sopsFile = ../../../../secrets/ssh-hosts.conf;
        format = "binary";
        path = "${config.home.homeDirectory}/.ssh/config.inventory";
      };
    };
  };
}
