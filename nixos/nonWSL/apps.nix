{
    lib,
    ...
}: {
    nixpkgs.config.allowUnfree = lib.mkForce true;
    programs._1password-gui = {
        enable = true;
        polkitPolicyOwners = [ "lemonilemon" ];
    };
}
