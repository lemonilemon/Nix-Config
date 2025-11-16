{
  lib,
  config,
  pkgs,
  username,
  ...
}:
{
  config = lib.mkIf config.nixos.gui.development.web.enable {
    programs.wireshark = {
      enable = true; # Install Wireshark
      package = pkgs.wireshark;
      dumpcap.enable = true;
    };
    users.users.${username}.extraGroups = [ "wireshark" ]; # Allow user to capture packets
  };
}
