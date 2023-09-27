{ pkgs, ... }:
{
    environment.systemPackages = with pkgs; [
        gcc # c/c++
        python3 # python
        nodejs # nodejs
        texlive.combined.scheme-medium # tex
    ]; 
}
