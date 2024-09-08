{
    programs.nixvim = {
        plugins = {
            # https://nix-community.github.io/nixvim/plugins/notify/index.html
            notify = {
                enable = true;
                backgroundColour = "#000000";
                fps = 60;
                render = "default";
                timeout = 1000;
                topDown = true;
            };
        };
    };
}
