{
  inputs,
  ...
}:
[
  # AI coding agents — use the flake package set directly so store paths match
  # the artifacts served from cache.numtide.com.
  (final: _prev: {
    llm-agents = inputs.llm-agents.packages.${final.stdenv.hostPlatform.system};
  })
]
