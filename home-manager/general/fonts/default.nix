{
    pkgs,
    ...
}: {
    fonts.fontconfig.enable = true;
    home.packages = with pkgs; [
        noto-fonts
        noto-fonts-cjk
        noto-fonts-emoji
        (nerdfonts.override { fonts = [ "FiraCode" "DroidSansMono" ]; })
    ];
}
