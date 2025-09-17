{
  lib,
  config,
  pkgs,
  username,
  ...
}:
{
  config = lib.mkIf config.home.gui-settings.development.web.enable {
    home.packages = with pkgs; [
      postman # API development tool
      wireshark # Network protocol analyzer
      tcpdump # Command-line packet analyzer
      nmap # Network discovery tool
    ];
  };
}
