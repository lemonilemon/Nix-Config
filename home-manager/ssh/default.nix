{
    ...
}: {
    programs.ssh = {
        enable = true;
        extraConfig = let onePassPath = "~/.1password/agent.sock"; in ''
            Host *
                IdentityAgent ${onePassPath}
        '';
    };
}
