{
    ...
}: {
    imports = [
        ./options/opts.nix
        ./options/keymaps.nix
        ./options/colorschemes.nix
    ];
    programs.nixvim = {
        enable = true;
        defaultEditor = true;
        viAlias = true;
        vimAlias = true;
        vimdiffAlias = true;
    };
}
