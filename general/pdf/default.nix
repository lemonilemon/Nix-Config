{ pkgs, ... }:
{
    environment.systemPackages = with pkgs; [
        # tex & markdown
        texlive.combined.scheme-medium # tex
        multimarkdown # markdown
    ]; 
}
