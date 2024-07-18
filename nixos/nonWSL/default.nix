{
    config,
    lib,
    ...
}: with lib;
let 
    cfg = config.programs.nonWSLapps; 
    mkDisableOption = name: mkEnableOption name // {
        default = false;
        example = true;
    };
in {
    imports = [
        ./displayManager
        ./apps.nix
    ];
    options.programs.nonWSLapps = {
        enable = mkDisableOption "non-WSL application";
    };
}
