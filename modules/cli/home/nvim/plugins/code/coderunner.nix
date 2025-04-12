{
  pkgs,
  ...
}:
{
  programs.nixvim = {
    extraPlugins = with pkgs.vimUtils; [
      (buildVimPlugin {
        pname = "CodeRunner";
        version = "2025-04-11";
        src = pkgs.fetchFromGitHub {
          owner = "CRAG666";
          repo = "code_runner.nvim";
          rev = "main";
          sha256 = "sha256-TOMxkVd6jL76VVrUhUuu7KdC+D2iHk9Dchrdbohj4VY=";
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
