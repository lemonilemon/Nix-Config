{
  nixpkgs.overlays = [
    # Use no-remote as default (firefox)
    (final: prev: {
      firefox = prev.firefox.overrideAttrs (oldAttrs: {
        buildCommand = ''
          ${oldAttrs.buildCommand}
          mv $out/bin/firefox $out/bin/firefox-bin
          cat > $out/bin/firefox <<EOF
          #!/bin/sh
          exec $out/bin/firefox-bin -no-remote "\$@"
          EOF
          chmod +x $out/bin/firefox
        '';
      });
    })
  ];
}
