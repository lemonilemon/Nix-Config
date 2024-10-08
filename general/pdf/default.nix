{ pkgs, ... }:
{
  environment.systemPackages = with pkgs; [
    # tex & markdown
    texlive.combined.scheme-full # tex
    multimarkdown # markdown

    # file conversion
    pandoc

    # pdf reader
    zathura
  ];
}
