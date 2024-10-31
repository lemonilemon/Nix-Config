{
  lib,
  ...
}:
{
  programs.starship = {
    enable = true;
    settings = {
      format = lib.concatStrings [
        "$os"
        "$all"
      ];
      right_format = lib.concatStrings [
        "$battery"
        "$time"
      ];
      "$schema" = "https://starship.rs/config-schema.json";
      palette = "catppuccin_mocha";
      # Some specific configurations of the prompt format and style
      character = {
        success_symbol = "[[󰄛](green) ❯](peach)";
        error_symbol = "[[󰄛](red) ❯](peach)";
        vimcmd_symbol = "[󰄛 ❮](subtext1)"; # For use with zsh-vi-mode
      };
      time = {
        utc_time_offset = "local";
        disabled = false;
      };
      git_branch = {
        style = "bold mauve";
      };
      os = {
        style = "bold sky";
        disabled = false;
      };
      battery.display = {
        threshold = 100;
      };
      directory = {
        truncation_length = 4;
        style = "bold lavender";
      };
      git_status = {
        ahead = "⇡$\{count}";
        diverged = "⇕⇡$\{ahead_count}⇣$\{behind_count}";
        behind = "⇣$\{count}";
      };

      # The color palette for the prompt, the default one is "mocha"
      palettes.catppuccin_mocha = {
        rosewater = "#f5e0dc";
        flamingo = "#f2cdcd";
        pink = "#f5c2e7";
        mauve = "#cba6f7";
        red = "#f38ba8";
        maroon = "#eba0ac";
        peach = "#fab387";
        yellow = "#f9e2af";
        green = "#a6e3a1";
        teal = "#94e2d5";
        sky = "#89dceb";
        sapphire = "#74c7ec";
        blue = "#89b4fa";
        lavender = "#b4befe";
        text = "#cdd6f4";
        subtext1 = "#bac2de";
        subtext0 = "#a6adc8";
        overlay2 = "#9399b2";
        overlay1 = "#7f849c";
        overlay0 = "#6c7086";
        surface2 = "#585b70";
        surface1 = "#45475a";
        surface0 = "#313244";
        base = "#1e1e2e";
        mantle = "#181825";
        crust = "#11111b";
      };
      # With nerd-fonts
      aws.symbol = "  ";
      buf.symbol = " ";
      c.symbol = " ";
      conda.symbol = " ";
      crystal.symbol = " ";
      dart.symbol = " ";
      directory.read_only = " 󰌾";
      docker_context.symbol = " ";
      elixir.symbol = " ";
      elm.symbol = " ";

      fennel.symbol = " ";

      fossil_branch.symbol = " ";

      git_branch.symbol = " ";

      git_commit.tag_symbol = "  ";

      golang.symbol = " ";

      guix_shell.symbol = " ";

      haskell.symbol = " ";

      haxe.symbol = " ";

      hg_branch.symbol = " ";

      hostname.ssh_symbol = " ";

      java.symbol = " ";
      julia.symbol = " ";

      kotlin.symbol = " ";

      lua.symbol = " ";

      memory_usage.symbol = "󰍛 ";

      meson.symbol = "󰔷 ";

      nim.symbol = "󰆥 ";

      nix_shell.symbol = " ";

      nodejs.symbol = " ";

      ocaml.symbol = " ";

      os.symbols = {
        Alpaquita = " ";
        Alpine = " ";
        AlmaLinux = " ";
        Amazon = " ";
        Android = " ";
        Arch = " ";
        Artix = " ";
        CentOS = " ";
        Debian = " ";
        DragonFly = " ";
        Emscripten = " ";
        EndeavourOS = " ";
        Fedora = " ";
        FreeBSD = " ";
        Garuda = "󰛓 ";
        Gentoo = " ";
        HardenedBSD = "󰞌 ";
        Illumos = "󰈸 ";
        Kali = " ";
        Linux = " ";
        Mabox = " ";
        Macos = " ";
        Manjaro = " ";
        Mariner = " ";
        MidnightBSD = " ";
        Mint = " ";
        NetBSD = " ";
        NixOS = " ";
        OpenBSD = "󰈺 ";
        openSUSE = " ";
        OracleLinux = "󰌷 ";
        Pop = " ";
        Raspbian = " ";
        Redhat = " ";
        RedHatEnterprise = " ";
        RockyLinux = " ";
        Redox = "󰀘 ";
        Solus = "󰠳 ";
        SUSE = " ";
        Ubuntu = " ";
        Unknown = " ";
        Void = " ";
        Windows = "󰍲 ";
      };

      package.symbol = "󰏗 ";

      perl.symbol = " ";

      php.symbol = " ";

      pijul_channel.symbol = " ";

      python.symbol = " ";

      rlang.symbol = "󰟔 ";

      ruby.symbol = " ";

      rust.symbol = "󱘗 ";

      scala.symbol = " ";

      swift.symbol = " ";

      zig.symbol = " ";

      gradle.symbol = " ";
    };
  };
}
