{
  username,
  ...
}: {
    home.username = "${username}";
    home.homeDirectory = "/home/${username}";
    imports = [ 
        ./nvim
        ./shells
	    ./git
        ./programs
        ./zellij
    ];
    home.stateVersion = "23.05";
    programs.home-manager.enable = true;
    nixpkgs.config.allowUnfreePredicate = _: true;
}
