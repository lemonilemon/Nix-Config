{ 
    ... 
}: {
    programs.git = {
        enable = true;
        userName = "lemonilemon";
        userEmail = "imlemonilemon@gmail.com";
        init = {
            defaultBranch = "main";
        };
    };
}
