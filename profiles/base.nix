{
  ...
}:
{
  nixpkgs.config = {
    allowUnfree = true;
    allowBroken = true;
  };
  hardware.graphics.enable32Bit = true;
}
