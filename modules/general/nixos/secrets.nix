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
  # The file lives under /home, which is a btrfs subvolume of the root device
  # on both physical hosts -- if it were ever a separate volume that mounts
  # late, this would need to move to /var/lib/sops-nix instead.
  config = lib.mkIf config.nixos.general.enable {
    sops.age.keyFile = "/home/${username}/.config/sops/age/keys.txt";
  };
}
