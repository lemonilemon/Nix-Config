{
  lib,
  ...
}:
{
  # Sound
  services.pulseaudio.enable = false; # Use pipewire instead
  security.rtkit.enable = false;
  services.pipewire = {
    enable = true;
    alsa.enable = true;
    alsa.support32Bit = true;
    pulse.enable = true;
    jack.enable = true;
    # screensharing
    wireplumber.enable = true;
    extraConfig.pipewire."10-stable-config" = {
      context.properties = {
        "log.level" = 0;
        "default.clock.rate" = 48000;
        "default.clock.quantum" = 1024; # Larger quantum = more stable
        "default.clock.min-quantum" = 512;
        "default.clock.max-quantum" = 2048;
        "core.daemon" = true;
        "core.name" = "pipewire-0";
        "mem.warn-mlock" = false;
        "mem.allow-mlock" = false; # Disable memory locking
      };
    };
  };
  # Clean environment
  environment.sessionVariables = {
    "PIPEWIRE_LATENCY" = "1024/48000";
  };
}
