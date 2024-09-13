;; tex
\documentclass[12pt, a4paper, UTF8]{article}
\usepackage[margin=1in]{geometry}
\usepackage{algorithm2e}
\usepackage{fancyhdr}
\usepackage[shortlabels]{enumitem}
\usepackage{CJKutf8}
\usepackage[all]{xy}
\usepackage{systeme}

\usepackage{amsmath,amsthm,amssymb,color,latexsym}
\usepackage{geometry}        
\geometry{letterpaper}    
\RestyleAlgo{ruled}
\usepackage{graphicx}
\graphicspath{ {./images/} }
\LinesNumbered

\newlist{subprobs}{enumerate}{2}
\setlist[subprobs, 1] {   
    label = (\alph{subprobsi}),
    leftmargin=\parindent,
    rightmargin=10pt
}
\setlist[subprobs, 2] {
    label = (\alph{subprobsi}-\arabic{subprobsii}),
    leftmargin=\parindent,
    rightmargin=10pt
}


\newenvironment{solution}[1][\it{Solution}]{\textbf{#1. }\\}{\begin{flushright}$\square$\end{flushright}}
\newenvironment{claim}[1][\it{Claim}]{
    \begin{center}
        \begin{tabular}{|p{0.9\textwidth}|}
    \hline\\
    \textbf{#1. }\\
}{
    \\\\\hline
        \end{tabular}
    \end{center}
}

\begin{document}
% headers and footers
\pagestyle{fancy}
\fancyhead{} % clear all header fields
\fancyhead[R]{}
\fancyhead[L]{}
\fancyfoot{} % clear all footer fields
\fancyfoot[C]{\thepage}
\fancyfoot[L]{\begin{CJK}{UTF8}{gkai} Name: {{_author_}}\end{CJK}}
\fancyfoot[R]{Student ID: }

\end{document}
