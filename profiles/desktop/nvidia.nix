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
        powerManagement.enable = true;
        nvidiaSettings = true;
        open = true; # enable the open source NVIDIA kernel module.
      };
      nvidia-container-toolkit.enable = true; # NVIDIA container toolkit
    };
    # Load nvidia driver for Xorg and Wayland
    services.xserver.videoDrivers = [ "nvidia" ];
    # Early KMS: the NVIDIA stack in the initrd so Plymouth can take over the
    # display during early boot. Grows the initrd by tens of MB (zstd helps).
    boot.initrd.kernelModules = [
      "nvidia"
      "nvidia_modeset"
      "nvidia_uvm"
      "nvidia_drm"
    ];
    environment.sessionVariables = {
      CUDA_PATH = "${pkgs.cudatoolkit}"; # CUDA toolkit path
    };
  };
}
