{
    lib,
    ...
}: {
    nixpkgs.config.allowUnfree = lib.mkDefault true;
    imports = [
	    ./shells
        ./programlangs
        ./nix
        ./pdf
        ./utils
        ./fonts
        ./secrets
    ];
}
