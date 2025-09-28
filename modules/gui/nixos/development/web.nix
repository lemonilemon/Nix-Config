{
  pkgs,
  username,
  ...
}:
{
  programs.wireshark = {
    enable = true; # Install Wireshark
    package = pkgs.wireshark;
    dumpcap.enable = true;
  };
  users.users.${username}.extraGroups = [ "wireshark" ]; # Allow user to capture packets
}
