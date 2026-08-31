{
  config,
  lib,
  username,
  ...
}:
{
  # Where sops-nix finds the age key when decrypting *system* secrets. The
  # Home Manager side sets its own copy of this in gui/home/apps; this is the
  # NixOS twin, and it points at the same key file so there is one key to back
  # up rather than two.
  #
  # Deliberately not behind a leaf feature flag of its own. It is the key
  # source for the whole category rather than a feature, and a flag whose only
  # possible effect is to make every system secret fail to decrypt is a trap
  # rather than a knob. sops-nix's entire config block is gated on
  # `sops.secrets != {}`, so on a host that declares no secrets this line
  # costs nothing.
  #
  # The key may live on any filesystem, including one that mounts in stage 2:
  # `useSystemdActivation` below is what makes that safe. Three separate
  # readers want this exact path -- this module as root, the Home Manager side
  # as the user, and the `sops` CLI when editing secrets/ by hand -- so it
  # stays one user-owned file rather than being moved or copied to
  # /var/lib/sops-nix, which only the first of the three could read.
  config = lib.mkIf config.nixos.general.enable {
    sops.age.keyFile = "/home/${username}/.config/sops/age/keys.txt";

    # Install secrets from a stage-2 systemd unit instead of an activation
    # script snippet.
    #
    # `boot.initrd.systemd.enable` (profiles/boot.nix) moves activation into
    # the initrd, where only / and /nix are mounted. The key above sits on
    # /home -- a separate mount unit that lands about two seconds later in
    # stage 2 -- so the activation script could never read it: every boot
    # failed `setupSecrets`, left /run/secrets absent entirely, and took down
    # anything naming a secret in its ExecStartPre.
    #
    # Sharing a device with / is not what would save you there; being mounted
    # in stage 1 is. That makes `neededForBoot` on /home the obvious other
    # fix, and a poor one -- it promotes a home directory to a boot blocker to
    # work around an ordering bug. This removes the ordering problem instead:
    # sops-nix derives `RequiresMountsFor=` from `age.keyFile`, so the unit
    # waits on whichever mount backs the key, wherever that key later moves.
    sops.useSystemdActivation = true;

    # sops-nix gives that unit `WantedBy=sysinit.target`, which pulls it into
    # early boot but orders nothing against it -- WantedBy is a requirement
    # edge, not an ordering edge. Its only `Before=` is
    # sysinit-reactivation.target, which covers `nixos-rebuild switch` rather
    # than boot. So a service reading /run/secrets races the oneshot instead
    # of waiting for it, and merely tends to win because installing secrets is
    # fast.
    #
    # Ordering it before sysinit.target restores what the activation script
    # gave for free: secrets are in place before any ordinary unit starts.
    # The resulting chain, home.mount -> local-fs.target -> here ->
    # sysinit.target, is acyclic.
    systemd.services.sops-install-secrets.before = [ "sysinit.target" ];
  };
}
