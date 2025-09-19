{
  username,
  ...
}:
{
  services.samba = {
    enable = true;
    openFirewall = true;

    # Use settings instead of extraConfig
    settings = {
      global = {
        workgroup = "WORKGROUP";
        "server string" = "NixOS File Server";
        "netbios name" = "nixos-server";
        security = "user";
        "map to guest" = "bad user";

        # Better compatibility with various devices
        "client min protocol" = "SMB2";
        "server min protocol" = "SMB2";
      };

      # Public share (no authentication)
      public = {
        path = "/home/${username}/Public";
        browseable = "yes";
        "read only" = "no";
        "guest ok" = "yes";
        "create mask" = "0664";
        "directory mask" = "0775";
        "force user" = "${username}";
        "force group" = "users";
      };

      # Private share (requires login)
      private = {
        path = "/home/${username}/Private";
        browseable = "yes";
        "read only" = "no";
        "valid users" = "${username}";
        "create mask" = "0664";
        "directory mask" = "0775";
      };

      # Media share (good for streaming to TVs)
      media = {
        path = "/home/${username}/Media";
        browseable = "yes";
        "read only" = "yes";
        "guest ok" = "yes";
        "force user" = "${username}";
      };
    };
  };

  # Create the directories
  systemd.tmpfiles.rules = [
    "d /home/${username}/Public 0775 ${username} users -"
    "d /home/${username}/Private 0775 ${username} users -"
    "d /home/${username}/Media 0775 ${username} users -"
  ];

  # If you want authenticated access, set up Samba users
  # Run after rebuild: sudo smbpasswd -a ${username}
}
