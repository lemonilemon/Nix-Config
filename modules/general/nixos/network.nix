{
  pkgs,
  lib,
  config,
  ...
}:
let
  cfg = config.nixos.general.network;
in
{
  config = lib.mkMerge [
    {
      # Network issues with wpa_supplicant
      systemd.services.wpa_supplicant.environment.OPENSSL_CONF = pkgs.writeText "openssl.cnf" ''
        openssl_conf = openssl_init
        [openssl_init]
        ssl_conf = ssl_sect
        [ssl_sect]
        system_default = system_default_sect
        [system_default_sect]
        Options = UnsafeLegacyRenegotiation
        [system_default_sect]
        CipherString = Default:@SECLEVEL=0
      '';
    }

    (lib.mkIf cfg.enable {
      # NetworkManager runs its own supplicant; the standalone service conflicts.
      networking.wireless.enable = lib.mkDefault false;

      networking.networkmanager = lib.mkIf cfg.manager.enable {
        enable = true;
        plugins = with pkgs; [
          networkmanager-openvpn # OpenVPN support
          networkmanager-openconnect # OpenConnect support
        ];
        wifi.powersave = lib.mkDefault cfg.wifi.enable;
      };

      hardware.enableRedistributableFirmware = lib.mkDefault cfg.firmware.enable;
    })
  ];
}
