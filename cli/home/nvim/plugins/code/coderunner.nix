{
  pkgs,
  ...
}:
{
  programs.nixvim = {
    extraPlugins = with pkgs.vimUtils; [
      (buildVimPlugin {
        pname = "CodeRunner";
        version = "2024-09-08";
        src = pkgs.fetchFromGitHub {
          owner = "CRAG666";
          repo = "code_runner.nvim";
          rev = "main";
          sha256 = "sha256-ibYwGZ49/NSn5PbN4gjWiiPSkdbqL9koN4GTn7/UIzY=";
        };
      })
    ];
    extraConfigLua = ''
      require('code_runner').setup({
          filetype = {
              python = "python3 -u",
              typescript = "deno run",
              rust = {
                  "cd $dir &&",
                  "rustc $fileName &&",
                  "$dir/$fileNameWithoutExt"
              },
              c = {
                  "cd $dir &&",
                  "gcc $fileName -o $fileNameWithoutExt.out &&",
                  "$dir/$fileNameWithoutExt.out &&",
                  "rm $dir/$fileNameWithoutExt.out",
              },
              cpp = {
                  "cd $dir &&",
                  "g++ $fileName -o $fileNameWithoutExt.out &&",
                  "$dir/$fileNameWithoutExt.out &&",
                  "rm $dir/$fileNameWithoutExt.out",
              },
              tex = function()
                  vim.api.nvim_command("VimtexCompile");
              end,
              markdown = function()
                  vim.api.nvim_command("MarkdownPreview");
              end,
          }
      })
    '';
  };
}
