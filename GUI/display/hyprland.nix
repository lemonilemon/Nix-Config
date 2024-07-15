{
    pkgs,
    ...
}: {
    environment.systemPackages = [
        pkgs.kitty
    ];
    programs.hyprland = {
        enable = true;
        xwayland.enable = true;
    };
}
