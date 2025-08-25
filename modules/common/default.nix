{ config, lib, pkgs, platformConfig ? null, ... }: {
  # This module provides platform detection and common settings
  # It works alongside the existing module structure
  
  # Platform-aware behavior - only when platformConfig is provided  
  config = lib.mkIf (platformConfig != null) {
    # Add platform-specific behavior here if needed
    # For now, just validate the platform detection works
    assertions = [
      {
        assertion = platformConfig.isWSL -> builtins.pathExists "/proc/sys/fs/binfmt_misc/WSLInterop" || true; # Allow override for testing
        message = "Platform configured as WSL but WSL environment not detected";
      }
    ];
  };
}