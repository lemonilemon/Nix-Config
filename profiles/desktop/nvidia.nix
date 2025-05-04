{
  pkgs,
  config,
  ...
}:
{
  config = {
    environment.systemPackages = with pkgs; [
      cudatoolkit
    ];
    nixpkgs.config.cudaSupport = true;
    nixpkgs.config.nvidia.acceptLicense = true;
    hardware = {
      graphics.enable = true;
      enableRedistributableFirmware = true;
      nvidia = {
        modesetting.enable = true;
        package = config.boot.kernelPackages.nvidiaPackages.stable;

        # prime = {
        #   sync.enable = true;
        # };
        nvidiaSettings = true;
        open = true; # enable the open source NVIDIA kernel module.
      };
      nvidia-container-toolkit.enable = true; # NVIDIA container toolkit
    };
    # Load nvidia driver for Xorg and Wayland
    services.xserver.videoDrivers = [ "nvidia" ];
    environment.sessionVariables = {
      CUDA_PATH = "${pkgs.cudatoolkit}"; # CUDA toolkit path
    };
  };
}
