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
    ];
    home.stateVersion = "23.05";
    programs.home-manager.enable = true;
    nixpkgs.config.allowUnfreePredicate = _: true;
}
