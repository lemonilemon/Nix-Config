{
  config,
  lib,
  username,
  ...
}:
let
  cfg = config.nixos.general.syncthing;

  # The vault's location is a user-level fact shared with obsidian.nvim and the
  # Obsidian package, so it is declared once under `home.general.obsidian` --
  # which this repo declares in the NixOS tree too -- rather than duplicated as
  # a syncthing-specific path that could drift away from the editor's.
  vault = config.home.general.obsidian;

  # The vault mesh, keyed by the name a host answers to through
  # `nixos.general.syncthing.deviceName`.
  #
  # Device IDs are public -- each is a hash of the public half of that device's
  # certificate, and Syncthing prints them in its own UI -- so they belong in
  # git. The private halves are the secret and live in secrets/syncthing.yaml.
  # Safe in a *public* repo specifically because global discovery is off below;
  # an ID is only a privacy problem when there is an address record to resolve
  # it against.
  # Committing the IDs is what removes the pairing round-trip: a reinstalled
  # host restores its key from sops and is immediately the same device to
  # everyone else, rather than a stranger waiting to be accepted.
  #
  # `phone` is the Android device (Syncthing-Fork). Unlike the two NixOS hosts
  # it has no entry in secrets/syncthing.yaml: its keypair is generated inside
  # the app's private storage and cannot be provisioned from here, so it is
  # only ever a peer, never a `deviceName`. Reinstalling the app gives it a new
  # identity and this row then has to be updated by hand.
  #
  # A null row is dropped rather than emitted, so a device can be parked here
  # before its ID is known without producing a half-configured peer.
  #
  # Checksum-verified on entry: all four Luhn check digits of each ID above
  # validate, so none of them is a transcription error.
  mesh = {
    desktop = "H2Z5BBK-NCRAM5P-LHRLZSZ-WWYXYAN-3EHKWAE-BR226WT-EMI5FXJ-MJI5AAW";
    laptop = "H6C5IP4-6D2EWOJ-7373PHU-K2P5L4I-IC5ZR4S-X72HOXL-KNZU65I-BIBVHAK";
    phone = "BH2TVXF-IXSL3FU-XSYVFVS-73JWIGH-3APWVI6-AH4TXXN-VG7M5NA-TDQNQQI";
  };

  peers = lib.filterAttrs (name: id: id != null && name != cfg.deviceName) mesh;

  # Version snapshots live outside the vault on purpose. Syncthing's default
  # puts them in `<folder>/.stversions`, and the vault is a git working tree --
  # a directory of every previous revision of every note, sitting inside a repo
  # that auto-commits, is a footgun with no upside. `fsPath` moves them out
  # entirely, so git never has an opinion about them.
  versionsDir = "/home/${username}/.local/state/syncthing/versions/obsidian-notes";

  # Syncthing and git share one working tree here, so this list may only name
  # paths the vault's ignore rules already cover. An ignore over a *tracked*
  # file would arrive on the other machine as a deletion, and the vault's
  # "vault backup" commits would then record it as one.
  #
  # Verified against the vault on 2026-08-30: none of these are tracked, and
  # each is either per-machine UI state or a regenerable cache.
  #
  # `.obsidian/plugins/**` is deliberately absent. Git ignores the plugin code
  # (keeping only each plugin's data.json), but syncing it is precisely how the
  # second machine ends up with the same plugins installed.
  ignorePatterns = [
    ".smart-env" # Smart Connections embedding cache: 14M, rewritten constantly
    ".obsidian/workspace.json" # which panes are open, per machine
    ".obsidian/workspace-mobile.json"
    ".obsidian/graph.json" # graph view camera position, per machine
    ".tmp" # local extraction cache
    ".trash" # Obsidian's own trash, per machine
  ];
in
{
  config = lib.mkIf cfg.enable {
    # Only this host's own identity is decrypted; the other hosts' keys stay
    # ciphertext in the same file. Left root-owned (sops' default 0400): the
    # syncthing unit's ExecStartPre is a `+`-prefixed privileged step, so it
    # reads these as root and installs user-owned copies into configDir.
    sops.secrets = {
      "syncthing-cert" = {
        sopsFile = ../../../secrets/syncthing.yaml;
        key = "${cfg.deviceName}/cert";
      };
      "syncthing-key" = {
        sopsFile = ../../../secrets/syncthing.yaml;
        key = "${cfg.deviceName}/key";
      };
    };

    services.syncthing = {
      enable = true;

      # Run as the human, not as a `syncthing` service user: the vault is a git
      # working tree the user edits directly, and a second uid writing into it
      # would leave files they cannot commit.
      user = username;
      group = username;
      dataDir = "/home/${username}";
      configDir = "/home/${username}/.config/syncthing";

      cert = config.sops.secrets."syncthing-cert".path;
      key = config.sops.secrets."syncthing-key".path;

      # The config in this file is the whole truth. Without these two, a device
      # or folder added by hand in the web UI would survive rebuilds, and the
      # repo would stop describing what the machine actually syncs.
      overrideDevices = true;
      overrideFolders = true;

      # TCP/UDP 22000 for transfers and UDP 21027 for discovery. These merge
      # with `nixos.general.firewall.allowedTCPPorts` rather than replacing it:
      # the underlying option is a listOf, so both definitions concatenate.
      openDefaultPorts = true;

      settings = {
        # Leaves the GUI on 127.0.0.1:8384, the module's default. Worth being
        # explicit about on the desktop, which trusts 192.168.0.0/24 wholesale
        # in its firewall -- binding loopback is what keeps the LAN bypass from
        # also handing out the admin interface.
        gui.address = "127.0.0.1:8384";

        options = {
          # -1 is an explicit "no" to anonymous usage reporting. The default, 0,
          # means "not asked yet", and Syncthing raises a UI nag later.
          urAccepted = -1;

          # LAN-only sync, and the reason the device IDs above are safe in a
          # public repo.
          #
          # Global discovery announces "device X is reachable at these
          # addresses" to discovery.syncthing.net, where lookups are
          # *unauthenticated* -- Syncthing's query client deliberately presents
          # no certificate. So anyone holding a device ID can resolve it to
          # that machine's public IP, LAN addresses and online status.
          # Committing the IDs with this on would hand that key to everyone.
          #
          # Switching the announcement off removes the record instead of hiding
          # the key to it: with nothing published, a known ID resolves to
          # nothing. Local discovery is a separate mechanism -- a UDP broadcast
          # on 21027 that never leaves the network segment -- and is what
          # actually finds peers here. It stays at its default of true.
          #
          # The cost is the honest one: devices sync when they share a network.
          # For off-LAN sync later, either flip these back on, or give each
          # device a static address alongside "dynamic" (the address list is
          # resolved per-entry, so the two compose).
          globalAnnounceEnabled = false;
          relaysEnabled = false;
        };

        devices = lib.mapAttrs (name: id: { inherit id; }) peers;

        folders."obsidian-notes" = {
          # The folder ID is the join key across devices and must not change:
          # editing it makes every peer treat this as a brand-new folder.
          id = "obsidian-notes";
          label = "Obsidian notes";
          path = vault.vaultPath;
          devices = lib.attrNames peers;
          inherit ignorePatterns;

          versioning = {
            type = "staggered";
            fsPath = versionsDir;
            params = {
              # Hourly for a day, daily for a month, weekly until a year old.
              maxAge = toString (365 * 24 * 60 * 60);
              cleanInterval = "3600";
            };
          };
        };
      };
    };

    # Syncthing creates the folder itself on a receiving host, but not the
    # parents of an out-of-tree versions path. Both are declared here so the
    # first sync on a fresh machine has somewhere to land.
    systemd.tmpfiles.rules = [
      "d ${vault.vaultPath} 0755 ${username} ${username} -"
      "d ${versionsDir} 0755 ${username} ${username} -"
    ];
  };
}
