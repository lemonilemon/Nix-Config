{
pkgs,
...
}: {
    services.xserver = {
        enable = true; 
        displayManager.gdm.enable = true;
        desktopManager.gnome.enable = true;
    };
    environment.gnome.excludePackages = (with pkgs; [
        gnome-tour
        gedit # text-editor
        gnome-terminal # terminal
        epiphany # web browser
        geary # email reader
    ]) ++ (with pkgs.gnome; [
        tali # poker game
        iagno # go game
        hitori # sudoku game
        atomix # puzzle game
    ]);
}
