{
  lib,
  ...
}:
{
  imports = [
    ./cli
    ./gui
    ./general
    ./desktop
    ./options.nix
  ];
  # formFactor lives here, not in modules/options.nix, on purpose: that file is
  # imported by both the NixOS tree (this file) and the Home Manager tree
  # (modules/home.nix), and formFactor only has meaning on the NixOS side.
  # Declaring it only in the NixOS-tree entrypoint makes `config.formFactor` a
  # hard eval error from Home Manager modules instead of a silent, wrong
  # "desktop" default on every host. Form-factor-dependent HM defaults should
  # mirror the NixOS value through `helpers.mkHomeOpt`, never read
  # `formFactor` directly.
  options = {
    formFactor = lib.mkOption {
      type = lib.types.enum [
        "laptop"
        "desktop"
        "wsl"
      ];
      default = "desktop";
      description = "Physical form factor of this host; seeds power, network, idle and bar defaults.";
    };
  };
  config = {
    catppuccin.enable = lib.mkDefault true;
    catppuccin.autoEnable = lib.mkDefault true;
    catppuccin.flavor = lib.mkDefault "mocha";
  };
}
