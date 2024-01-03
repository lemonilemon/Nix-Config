{
    pkgs,
    ...
}: {
    environment.systemPackages = with pkgs; [
        curl
        wget
        pkg-config
        dbus
        openssl_3
        glib
        gtk3
        libsoup
        webkitgtk
        librsvg
    ];
}
