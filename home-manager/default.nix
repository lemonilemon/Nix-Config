{
    username,
    ...
}: {
    home.username = "${username}";
    home.homeDirectory = "/home/${username}";
    imports = [ 
        ./general
        ./nvim
        ./shells
	    ./git
        ./programs
        ./zellij
        ./kitty
        ./ssh
    ];
    home.stateVersion = "23.05";
    programs.home-manager.enable = true;
}
