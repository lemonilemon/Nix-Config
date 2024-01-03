{
    pkgs,
    ...
}: {
    environment.systemPackages = with pkgs; [
        pkg-config
        pango
        glib
        gdk-pixbuf
    ];
}
