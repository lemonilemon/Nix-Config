{
    pkgs,
    ...
}: {
    environment.systemPackages = with pkgs; [

        # Terminal emulator
        kitty
        foot

        # Clipboard
        wl-clipboard

        # Notificatoin
        dunst
        libnotify

        # Wallpaper
        swww

        # Launcher
        rofi-wayland

        # System tray
        trayer

        # Data transfer
        socat

        # System information fetcher
        macchina

        # CLI utility to get icons
        geticons

        # Bar and widgets
        eww

        # Base Directory Specification
        xdg-desktop-portal-hyprland
    ];
    # Base Directory Specification
    xdg.portal= {
        enable = true;
        wlr.enable = true;
    };

    programs.hyprland = {
        enable = true;
        xwayland.enable = true;
    };

    environment.sessionVariables = {
        WLR_NO_HARDWARE_CURSORS = "1";
        NIXOS_OZONE_WL = "1";
    };
}
