{
  inputs,
  ...
}:
[
  # AI coding agents — updated daily, served from cache.numtide.com.
  # Use the shared-nixpkgs overlay so packages are built against
  # the host's nixpkgs version.
  inputs.llm-agents.overlays.shared-nixpkgs
]
