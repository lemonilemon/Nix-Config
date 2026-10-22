{
  lib,
  buildGoModule,
  replaceVars,
  writeText,
  runCommand,
  colors ? import ./palette.nix,
  enableLaptopControls ? false,
  enableBattery ? false,
  enableWifi ? false,
  defaultHidden ? [ ],
}:
let
  barModules = import ./bar-modules.nix;

  # The daemon, eww-barctl and eww-popup, one Go module.
  #
  # The daemon was Python until the port: 93% of its CPU is the child processes
  # its collectors fork (pactl, nmcli, hyprctl, ...), which cost the same in any
  # language, so the interpreter was never the expense. What moved it was the
  # CLIENTS -- eww.yuck invokes them from 39 handlers including :onscroll, and
  # they do almost no work: the daemon answers a ping in 0.06 ms against ~45 ms
  # of CPython startup spent asking it. Measured: 45.4 ms -> 3.3 ms for
  # eww-barctl, 50.8 ms -> 3.3 ms for eww-popup. Once those were compiled,
  # leaving the daemon behind meant maintaining the same collectors twice.
  backend = buildGoModule {
    pname = "eww-bar";
    version = "0";

    src = ./go;

    # Every import is stdlib, so there is nothing to vendor. vendorHash = null
    # creates no fetch derivation at all, which is the point: no hash to keep in
    # step with a dependency set that does not exist.
    vendorHash = null;

    # No subPackages: it would narrow checkPhase to cmd/, which holds no tests,
    # and silently skip every internal package. Building ./... installs the one
    # main package under cmd/ and runs `go test ./...` over the rest.

    postInstall = ''
      # argv[0] dispatch, as the Python entry point used to do: exec'ing a
      # symlink leaves argv[0] as the name the caller typed.
      ln -s eww-bar-client $out/bin/eww-barctl
      ln -s eww-bar-client $out/bin/eww-popup
    '';

    meta.mainProgram = "eww-bar-backend";
  };

  paletteScss = writeText "_palette.scss" (
    lib.concatStringsSep "\n" (
      lib.mapAttrsToList (name: value: "\$" + name + ": " + value + ";") colors
    )
  );

  yuck = replaceVars ./config/eww.yuck {
    laptopControls = if enableLaptopControls then "true" else "false";
    batteryModule = if enableBattery then "true" else "false";
    wifiControls = if enableWifi then "true" else "false";
    # defwindow takes a fixed size, and network_panel has nothing that expands,
    # so a window sized for the Wi-Fi row would leave a dead band inside the
    # card once that row is hidden. The row is ~40px including its margin.
    #
    # Still only two values after the speed card, which is the whole reason that
    # card renders in every state instead of appearing after a run: a
    # conditional card would have crossed with this and needed four.
    #
    # +114px over the original 200/160, measured rather than derived: the same
    # popup was rendered against a scratch eww daemon with and without the new
    # content, in a deliberately oversized window, and the footer's baseline
    # compared. GTK's natural heights do not agree with the arithmetic -- adding
    # up the min-heights and paddings under-predicted and would have clipped the
    # footer.
    #
    # 86 of it is the speed card and 28 the restrictions line. That line was a
    # row of SSH/DNS/IPv6 chips first, which cost 69px; folding it into the speed
    # card as one sentence gave 41px back and removed the legend nobody had.
    networkPopupHeight = if enableWifi then "314px" else "274px";
    barModulesDefault = builtins.toJSON (
      lib.genAttrs barModules (name: if lib.elem name defaultHidden then "false" else "true")
    );
  };
in
runCommand "eww-bar-${backend.version}"
  {
    passthru = { inherit backend yuck paletteScss; };
    meta.mainProgram = "eww-bar-backend";
  }
  ''
    mkdir -p $out/share/eww-bar
    ln -s ${backend}/bin $out/bin
    ln -s ${yuck} $out/share/eww-bar/eww.yuck
    ln -s ${./config/eww.scss} $out/share/eww-bar/eww.scss
    ln -s ${paletteScss} $out/share/eww-bar/_palette.scss
    ln -s ${./config/assets} $out/share/eww-bar/assets
  ''
