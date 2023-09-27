{
  username,
  ...
}: {
    home.username = "${username}";
    home.homeDirectory = "/home/${username}";
    imports = [ 
        ./nvim
        ./zsh
	    ./git
        ./programs
        ./tmux
    ];
    home.stateVersion = "23.05";
    programs.home-manager.enable = true;
}
