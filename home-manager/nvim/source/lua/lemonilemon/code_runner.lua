require('code_runner').setup({
    mode = "term",
    focus = true,
    startinsert = true,
    before_run_filetype = function()
        vim.cmd(":w")
    end,
    filetype = {
        java = {
            "cd $dir &&",
            "javac $fileName &&",
            "java $fileNameWithoutExt"
        },
        python = "python3 -u",
        c = {
            "cd $dir &&",
            "gcc $fileName -o $fileNameWithoutExt &&",
            "$dir/$fileNameWithoutExt &&",
            "rm $dir/$fileNameWithoutExt",
        },
        cpp = {
            "cd $dir &&",
            "g++ $fileName -o $fileNameWithoutExt &&",
            "$dir/$fileNameWithoutExt &&",
            "rm $dir/$fileNameWithoutExt",
        },
        typescript = "deno run",
        rust = {
            "cd $dir &&",
            "rustc $fileName &&",
            "$dir/$fileNameWithoutExt"
        },
        tex = function()
            require("code_runner.hooks.preview_pdf").run {
                command = "pdflatex",
                args = { "-output-directory", "/tmp", "$fileName" },
                preview_cmd = "zathura",
                overwrite_output = "/tmp",
            }
        end,
        markdown = function()
            require("code_runner.hooks.preview_pdf").run {
                command = "pandoc",
                args = { "$fileName", "-o", "$tmpFile", "-t", "pdf" },
                preview_cmd = "wsl-open > /dev/null",
                overwrite_output = "/tmp",
            }
        end,
    },
})
