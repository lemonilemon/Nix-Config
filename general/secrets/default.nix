{ 
    lib,
    ... 
}: {
    nixpkgs.config.allowUnfree = lib.mkForce true;

    programs._1password.enable = true;
}
