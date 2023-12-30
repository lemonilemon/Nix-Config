{ 
    ... 
}: {
    programs.git = {
        enable = true;
        userName = "lemonilemon";
        userEmail = "imlemonilemon@gmail.com";
        ignores = [ 
            "*.swp" 
            ".*"
        ];
        extraConfig = {
            core = {
                defaultBranch = "main";
            };
        };
    };
}
