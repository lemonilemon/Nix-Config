{ pkgs, ... }:
{
    environment.systemPackages = with pkgs; [
        gcc # c/c++

        # Python3
        python3
        python312Packages.pip
        virtualenv

        nodejs # nodejs
        jre8 # Java Development Kit
        rustc # Rust
        cargo # Rust projects
        # cargo-tauri # tauri
    ]; 
}
