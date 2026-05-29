{
  inputs,
  ...
}:
[
  # AI coding agents — updated daily, served from cache.numtide.com.
  # Use `default` (not `shared-nixpkgs`) so packages stay built against
  # the flake's pinned nixpkgs and hit the binary cache.
  inputs.llm-agents.overlays.default
]
