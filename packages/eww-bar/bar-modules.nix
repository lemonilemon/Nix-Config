# Bar modules in bar order. go/internal/collect/modules.json is the one list:
# the Go backend embeds it and Nix reads it here.
builtins.fromJSON (builtins.readFile ./go/internal/collect/modules.json)
