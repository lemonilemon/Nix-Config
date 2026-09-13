{
  config,
  lib,
  username,
  ...
}:
{
  config = lib.mkIf config.nixos.general.enable {
    # Must stay in step with the Home Manager twin in gui/home/apps and with
    # what the `sops` CLI reads when editing secrets/ by hand.
    sops.age.keyFile = "/home/${username}/.config/sops/age/keys.txt";

    sops.useSystemdActivation = true;
    systemd.services.sops-install-secrets.before = [ "sysinit.target" ];
  };
}
