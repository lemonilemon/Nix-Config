{
  username,
  ...
}:
let
  onePassPath = "~/.1password/agent.sock";
in
{
  programs._1password.enable = true;
  programs._1password-gui = {
    enable = true;
    polkitPolicyOwners = [ username ];
  };
  # programs.ssh = {
  #   extraConfig = ''
  #     Host *
  #         IdentityAgent ${onePassPath}
  #   '';
  # };
}
