{
  lib,
  config,
  pkgs,
  ...
}:
let
  hyprctl = "${pkgs.hyprland}/bin/hyprctl";
  hyprlock = "${pkgs.hyprlock}/bin/hyprlock";
  hyprshutdown = "${pkgs.hyprshutdown}/bin/hyprshutdown";
  systemctl = "${pkgs.systemd}/bin/systemctl";

  palette = import ../theme/palette.nix;

  # "#89b4fa" -> "137, 180, 250", so an accent can be emitted with an alpha.
  # Plain rgba() rather than GTK's alpha() helper: eww.scss records that a single
  # value GTK cannot parse makes it discard the rest of the stylesheet, and
  # rgba() is what has actually been verified against this binary.
  toRgb =
    hex:
    let
      digits = {
        "0" = 0;
        "1" = 1;
        "2" = 2;
        "3" = 3;
        "4" = 4;
        "5" = 5;
        "6" = 6;
        "7" = 7;
        "8" = 8;
        "9" = 9;
        "a" = 10;
        "b" = 11;
        "c" = 12;
        "d" = 13;
        "e" = 14;
        "f" = 15;
      };
      h = lib.toLower (lib.removePrefix "#" hex);
      byte = off: 16 * digits.${lib.substring off 1 h} + digits.${lib.substring (off + 1) 1 h};
    in
    "${toString (byte 0)}, ${toString (byte 2)}, ${toString (byte 4)}";

  rgba = hex: alpha: "rgba(${toRgb hex}, ${alpha})";

  # Order is load-bearing. wlogout's grid fills COLUMN-major -- gtk_grid_attach
  # is called with the buttons-per-row loop outermost -- so this sequence renders
  # as two rows that escalate left to right:
  #
  #   Lock       Logout   Suspend
  #   Hibernate  Reboot   Shutdown
  #
  # ending on the most destructive action rather than burying it mid-row.
  #
  # Glyphs are all Material Design Icons so the stroke weight is uniform; Lock is
  # md-lock-outline (U+F0341) rather than the filled md-lock, which rendered
  # visibly heavier than the other five.
  actions = [
    {
      name = "lock";
      glyph = "󰍁"; # U+F0341 md-lock-outline
      accent = palette.blue;
      label = "Lock";
      key = "l";
      action = "${hyprlock} -q";
    }
    {
      name = "hibernate";
      glyph = "󰜗"; # U+F0717 md-snowflake
      accent = palette.mauve;
      label = "Hibernate";
      key = "h";
      action = "${systemctl} hibernate";
    }
    {
      name = "logout";
      glyph = "󰍃"; # U+F0343 md-logout
      accent = palette.yellow;
      label = "Logout";
      key = "e";
      # The trailing hyprctl is a BACKSTOP, not the mechanism: hyprshutdown
      # exits Hyprland itself once the apps are closed, and only stops doing so
      # when passed --no-exit, which this does not. So the session normally ends
      # before that second command is reached, and this button kept working
      # through the 0.56 migration even though its old `dispatch exit` form was
      # by then a syntax error.
      #
      # Migrated anyway, because a backstop that cannot fire is not one. The
      # signature is from the shipped API stub, share/hypr/stubs/hl.meta.lua:
      # `exit fun(...)`, so no argument. Not exercised in testing, for the
      # obvious reason.
      action = "${hyprshutdown} && ${hyprctl} dispatch 'hl.dsp.exit()'";
    }
    {
      name = "reboot";
      glyph = "󰜉"; # U+F0709 md-restart
      accent = palette.peach;
      label = "Reboot";
      key = "r";
      action = "${systemctl} reboot";
    }
    {
      name = "suspend";
      glyph = "󰒲"; # U+F04B2 md-sleep
      accent = palette.teal;
      label = "Suspend";
      key = "u";
      action = "${systemctl} suspend";
    }
    {
      name = "shutdown";
      glyph = "󰐥"; # U+F0425 md-power
      accent = palette.red;
      label = "Shutdown";
      key = "s";
      action = "${systemctl} poweroff";
    }
  ];

  # wlogout draws its icons as CSS background-images, and a background-image
  # cannot inherit `color`. That is why the previous single mauve-tinted asset
  # per button could never take the button's accent: :hover recoloured the border
  # and the label and left the icon purple. Rasterising the glyph once per accent
  # is what makes hover reach the icon too.
  #
  # The art is the same Nerd Font the bar and hyprlock already use, so the stroke
  # weight matches by construction and no third-party icon set is vendored.
  # ImageMagick is handed the .ttf path directly, so the build needs no
  # fontconfig -- which is also what the old librsvg step existed to work around.
  icons =
    pkgs.runCommand "wlogout-icons"
      {
        nativeBuildInputs = [ pkgs.imagemagick ];
      }
      ''
        mkdir -p "$out"

        font=$(find ${pkgs.nerd-fonts.jetbrains-mono}/share/fonts \
                 -name 'JetBrainsMonoNerdFont-Regular.ttf' -print -quit)
        if [ -z "$font" ]; then
          echo "JetBrainsMonoNerdFont-Regular.ttf not found in nerd-fonts.jetbrains-mono" >&2
          exit 1
        fi

        render() { # $1=glyph  $2=colour  $3=outfile
          magick -size 512x512 xc:none -font "$font" -pointsize 380 -fill "$2" \
                 -gravity center -annotate +0+0 "$1" \
                 -trim +repage -background none -gravity center -extent 512x512 \
                 "$out/$3"
        }

        ${lib.concatMapStringsSep "\n" (a: ''
          render ${lib.escapeShellArg a.glyph} ${lib.escapeShellArg palette.text} ${a.name}-rest.png
          render ${lib.escapeShellArg a.glyph} ${lib.escapeShellArg a.accent} ${a.name}-accent.png
        '') actions}
      '';

  restRules = lib.concatMapStringsSep "\n" (a: ''
    #${a.name} {
      background-image: url("${icons}/${a.name}-rest.png");
      border-color: ${rgba a.accent "0.62"};
    }
  '') actions;

  hoverRules = lib.concatMapStringsSep "\n" (a: ''
    #${a.name}:hover {
      background-color: ${rgba a.accent "0.20"};
      background-image: url("${icons}/${a.name}-accent.png");
      border-color: ${a.accent};
      color: ${a.accent};
    }
  '') actions;
in
{
  imports = [
    ../hyprlock.nix
  ];
  config = lib.mkIf config.home.desktop.hyprland.wlogout.enable {
    # The catppuccin module prepends its own theme plus a set of mauve-tinted
    # SVGs for all six buttons. The rules below happen to override it, but only
    # by source order and only for properties that are restated here -- and its
    # theme also drops `* { background-image: none }` and `border-radius: 0` into
    # the cascade. Opting out makes this stylesheet the only one in play, exactly
    # as hyprlock.nix already does for its own theme.
    catppuccin.wlogout.enable = false;

    # Logout menu.
    #
    # This is a fullscreen session overlay, so it is styled as hyprlock's sibling
    # rather than as one of the eww bar's glass islands: the two tiers are
    # deliberately different. Values marked (hyprlock) are lifted from
    # hyprlock.nix's input-field so the pair reads as one family. The blurred
    # backdrop that completes the effect comes from the compositor -- see the
    # logout_dialog layerrule in ../default.nix.
    programs.wlogout = {
      enable = true;
      style = ''
        window {
          background-color: ${rgba palette.base "0.55"};
          background-image: none;
        }

        /* No container frame. hyprlock floats its one control straight on the
           blurred backdrop; these six do the same. */
        grid {
          background-color: transparent;
        }

        button {
          background-color: ${rgba palette.surface0 "0.80"}; /* (hyprlock) inner_color */
          background-position: center 34%;
          background-repeat: no-repeat;
          background-size: 25%;
          border-radius: 16px; /* = decoration.rounding */
          border-style: solid;
          border-width: 2px; /* (hyprlock) outline_thickness */
          box-shadow: none;
          color: ${palette.text}; /* (hyprlock) font_color */
          font-family: "JetBrains Mono Nerd Font"; /* (hyprlock) font_family */
          font-size: 17px;
          font-weight: 700;
          margin: 8px;
          /* The layout pins the label to the bottom of the tile (see `height`
             below), so this is what lifts it back up under the icon. Tile height
             differs by 60px between a 1200p and a 1080p output, which puts the
             label at 75% and 70% respectively -- a drift small enough to leave
             alone, and the reason no wrapper computes margins here. */
          padding-bottom: 76px;
          transition: background-color 200ms ease-out, border-color 200ms ease-out,
                      color 200ms ease-out;
        }

        button:focus {
          outline: none;
        }

        ${restRules}
        ${hoverRules}
      '';
      layout = map (a: {
        label = a.name;
        inherit (a) action;
        # The keybind is otherwise invisible. Baked in rather than using
        # wlogout's --show-binds, which strcats a bare "[l]" with no separator.
        text = "${a.label} · ${a.key}";
        keybind = a.key;
        # Pins gtk_label_set_yalign to 1.0 (bottom). wlogout allocates its button
        # array with malloc, never initialises yalign, and only ever assigns it
        # from this key -- so without it the label's vertical position is
        # whatever happened to be in that heap slot. It measured ~78% here, but
        # it is undefined and free to move on any wlogout bump, which would drop
        # the label onto the icon.
        #
        # Any value collapses to 1.0: the parser passes the first ASCII byte of
        # the number straight in as a float, so '1' becomes 49.0 and clamps. 1 is
        # written because it is what actually takes effect.
        height = 1;
      }) actions;
    };
  };
}
