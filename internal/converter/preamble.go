// preamble.go generates the LaTeX preamble used by pandoc to style the PDF output.
// The preamble is theme-aware: colors, fonts, heading styles, and callout colors
// all come from the active theme. This replaces the hardcoded LaTeX in the bash script.
package converter

import (
	"fmt"
	"strings"

	"github.com/jclement/pdfify/internal/config"
	"github.com/jclement/pdfify/internal/theme"
)

// GeneratePreamble builds a LaTeX preamble string from the given config and theme.
func GeneratePreamble(cfg *config.Config, t *theme.Theme) string {
	var b strings.Builder

	// --- Color definitions ---
	b.WriteString("% --- Theme: " + t.Name + " ---\n")
	b.WriteString("\\usepackage{xcolor}\n")
	writeColor(&b, "accent", t.Accent)
	writeColor(&b, "accentdark", t.AccentDark)
	writeColor(&b, "codebg", t.CodeBg)
	writeColor(&b, "codeborder", t.CodeBorder)
	writeColor(&b, "headrulecolor", t.HeadRuleColor)
	writeColor(&b, "titlebg", t.TitleBg)
	writeColor(&b, "tablerowgray", t.TableRowGray)

	// Callout colors
	writeColor(&b, "infobg", t.InfoBg)
	writeColor(&b, "infobar", t.InfoBar)
	writeColor(&b, "infofg", t.InfoFg)
	writeColor(&b, "tipbg", t.TipBg)
	writeColor(&b, "tipbar", t.TipBar)
	writeColor(&b, "tipfg", t.TipFg)
	writeColor(&b, "warningbg", t.WarningBg)
	writeColor(&b, "warningbar", t.WarningBar)
	writeColor(&b, "warningfg", t.WarningFg)
	writeColor(&b, "dangerbg", t.DangerBg)
	writeColor(&b, "dangerbar", t.DangerBar)
	writeColor(&b, "dangerfg", t.DangerFg)
	writeColor(&b, "examplebg", t.ExampleBg)
	writeColor(&b, "examplebar", t.ExampleBar)
	writeColor(&b, "examplefg", t.ExampleFg)
	writeColor(&b, "quotecallbg", t.QuoteBg)
	writeColor(&b, "quotecallbar", t.QuoteBar)
	writeColor(&b, "quotecallfg", t.QuoteFg)
	writeColor(&b, "linkcolor", t.LinkColor)

	// --- Code block styling ---
	b.WriteString(`
% --- Code block wrapping and styling ---
\usepackage{fvextra}
\DefineVerbatimEnvironment{Highlighting}{Verbatim}{
  breaklines,
  breakanywhere,
  commandchars=\\\{\},
  fontsize=\small
}

\usepackage[framemethod=tikz]{mdframed}

\makeatletter
\@ifundefined{Shaded}{\newenvironment{Shaded}{}{}}{}
\makeatother
\renewenvironment{Shaded}{%
  \begin{mdframed}[
    backgroundcolor=codebg,
    hidealllines=true,
    roundcorner=4pt,
    innertopmargin=8pt,
    innerbottommargin=8pt,
    innerleftmargin=10pt,
    innerrightmargin=10pt,
    skipabove=10pt,
    skipbelow=10pt
  ]
}{%
  \end{mdframed}
}

% --- Callout environments ---
\newenvironment{calloutbase}[3]{%
  \begin{mdframed}[
    backgroundcolor=#1,
    linecolor=#2,
    linewidth=3pt,
    topline=false,
    bottomline=false,
    rightline=false,
    innertopmargin=12pt,
    innerbottommargin=12pt,
    innerleftmargin=12pt,
    innerrightmargin=12pt,
    skipabove=12pt,
    skipbelow=12pt,
    roundcorner=0pt
  ]
  \textbf{\color{#2}#3}\par\smallskip\setlength{\parindent}{0pt}
}{%
  \end{mdframed}
}

\newenvironment{calloutinfo}[1]{\begin{calloutbase}{infobg}{infobar}{#1}}{\end{calloutbase}}
\newenvironment{callouttip}[1]{\begin{calloutbase}{tipbg}{tipbar}{#1}}{\end{calloutbase}}
\newenvironment{calloutwarning}[1]{\begin{calloutbase}{warningbg}{warningbar}{#1}}{\end{calloutbase}}
\newenvironment{calloutdanger}[1]{\begin{calloutbase}{dangerbg}{dangerbar}{#1}}{\end{calloutbase}}
\newenvironment{calloutexample}[1]{\begin{calloutbase}{examplebg}{examplebar}{#1}}{\end{calloutbase}}
\newenvironment{calloutquote}[1]{\begin{calloutbase}{quotecallbg}{quotecallbar}{#1}}{\end{calloutbase}}

% --- PDF bookmarks ---
\usepackage{bookmark}
\bookmarksetup{numbered=false, open, openlevel=2}

% --- Page break after TOC ---
\let\oldtableofcontents\tableofcontents
\renewcommand{\tableofcontents}{\oldtableofcontents\clearpage}

% --- TOC styling ---
\usepackage{tocloft}
\setlength{\cftbeforetoctitleskip}{0.5em}
\renewcommand{\cfttoctitlefont}{\LARGE\bfseries\color{accentdark}\scshape}
\renewcommand{\cftaftertoctitle}{\par\vspace{2pt}{\color{headrulecolor}\hrule height 1pt}\vspace{10pt}}
\renewcommand{\cftsecfont}{\bfseries\color{accentdark}}
\renewcommand{\cftsecpagefont}{\bfseries\color{accentdark}}
\renewcommand{\cftsubsecfont}{\color{accent}}
\renewcommand{\cftsubsecpagefont}{\color{accent}}
\renewcommand{\cftsubsubsecfont}{\small\color{accent}}
\renewcommand{\cftsubsubsecpagefont}{\small\color{accent}}
\renewcommand{\cftsecleader}{\cftdotfill{\cftsecdotsep}}
\renewcommand{\cftsecdotsep}{\cftdotsep}
\setlength{\cftbeforesecskip}{6pt}
\setlength{\cftbeforesubsecskip}{2pt}
`)

	// --- Heading font ---
	b.WriteString(fmt.Sprintf("\n%% --- Heading font ---\n"))
	b.WriteString(fmt.Sprintf("\\newfontfamily\\headingfont{%s}[BoldFont={%s Bold}]\n", t.HeadingFont, t.HeadingFont))

	// --- Symbol fallback ---
	b.WriteString(`
% --- Symbol fallback ---
\usepackage{newunicodechar}
\newfontfamily\fallbackfont{Liberation Sans}[Scale=MatchLowercase]
\newunicodechar{→}{{\fallbackfont →}}
\newunicodechar{←}{{\fallbackfont ←}}
\newunicodechar{↔}{{\fallbackfont ↔}}
\newunicodechar{⇒}{{\fallbackfont ⇒}}
\newunicodechar{⇐}{{\fallbackfont ⇐}}
\newunicodechar{✓}{{\fallbackfont ✓}}
\newunicodechar{✗}{{\fallbackfont ✗}}
`)

	// --- Section headings (theme-aware) ---
	b.WriteString("\n% --- Section headings ---\n")
	b.WriteString("\\usepackage{titlesec}\n\n")

	// H1 style
	scshape := ""
	if t.H1SmallCaps {
		scshape = "\\scshape"
	}
	b.WriteString(fmt.Sprintf("\\titleformat{\\section}\n  {\\LARGE\\headingfont\\bfseries\\color{accentdark}\\addfontfeatures{LetterSpace=%d}%s}\n  {\\thesection}{0.5em}{}[\\vspace{2pt}{\\color{headrulecolor}\\titlerule[1pt]}]\n", t.H1LetterSpace, scshape))
	b.WriteString("\\titlespacing*{\\section}{0pt}{20pt}{10pt}\n\n")

	// H2
	b.WriteString("\\titleformat{\\subsection}\n  {\\Large\\headingfont\\bfseries\\color{accentdark}\\addfontfeatures{LetterSpace=-1}}\n  {\\thesubsection}{0.5em}{}\n")
	b.WriteString("\\titlespacing*{\\subsection}{0pt}{16pt}{8pt}\n\n")

	// H3
	b.WriteString("\\titleformat{\\subsubsection}\n  {\\large\\bfseries\\color{accent}}\n  {\\thesubsubsection}{0.5em}{}\n")
	b.WriteString("\\titlespacing*{\\subsubsection}{0pt}{12pt}{6pt}\n\n")

	// H4
	b.WriteString("\\titleformat{\\paragraph}[hang]\n  {\\normalsize\\bfseries\\color{accent}}\n  {\\theparagraph}{0.5em}{}\n")
	b.WriteString("\\titlespacing*{\\paragraph}{0pt}{10pt}{4pt}\n\n")

	// --- Page style ---
	b.WriteString(`% --- Page style ---
\usepackage{fancyhdr}
\pagestyle{fancy}
\fancyhf{}
\renewcommand{\headrulewidth}{0pt}
\renewcommand{\footrulewidth}{0pt}
\setlength{\headheight}{14pt}
`)

	// Header
	if cfg.Header != "" {
		escaped := latexEscape(cfg.Header)
		b.WriteString(fmt.Sprintf("\\fancyhead[C]{\\color{gray}\\small %s}\n", escaped))
	}

	// Footer with page numbers
	b.WriteString("\\usepackage{lastpage}\n")
	if cfg.Footer != "" {
		escaped := latexEscape(cfg.Footer)
		b.WriteString(fmt.Sprintf("\\fancyfoot[L]{\\color{gray}\\small %s}\n", escaped))
	}
	b.WriteString("\\fancyfoot[R]{\\color{gray}\\small Page \\thepage\\ of \\pageref*{LastPage}}\n")

	// Plain page style (for title/TOC pages)
	b.WriteString("\\fancypagestyle{plain}{\\fancyhf{}\\renewcommand{\\headrulewidth}{0pt}\\renewcommand{\\footrulewidth}{0pt}")
	if cfg.Footer != "" {
		escaped := latexEscape(cfg.Footer)
		b.WriteString(fmt.Sprintf("\\fancyfoot[L]{\\color{gray}\\small %s}", escaped))
	}
	b.WriteString("\\fancyfoot[R]{\\color{gray}\\small Page \\thepage\\ of \\pageref*{LastPage}}}\n")

	// --- Blockquote styling ---
	b.WriteString(`
% --- Blockquote styling ---
\usepackage{etoolbox}
\renewenvironment{quote}{%
  \begin{mdframed}[
    backgroundcolor=infobg,
    linecolor=infobar,
    linewidth=3pt,
    topline=false,
    bottomline=false,
    rightline=false,
    innertopmargin=12pt,
    innerbottommargin=12pt,
    innerleftmargin=12pt,
    innerrightmargin=12pt,
    skipabove=10pt,
    skipbelow=10pt,
    roundcorner=0pt
  ]%
}{%
  \end{mdframed}%
}

% --- Table styling ---
\usepackage{booktabs}
\usepackage{colortbl}
\usepackage{longtable}
\usepackage{tabularx}
\arrayrulecolor{codeborder}

\definecolor{tablerowgray}{HTML}{` + t.TableRowGray + `}
\let\oldlongtable\longtable
\let\endoldlongtable\endlongtable
\renewenvironment{longtable}{\rowcolors{2}{white}{tablerowgray}\oldlongtable}{\endoldlongtable}

\usepackage{array}
\renewcommand{\arraystretch}{1.4}
\let\oldtexttt\texttt
\renewcommand{\texttt}[1]{{\small\oldtexttt{\seqsplit{#1}}}}
\usepackage{seqsplit}
\setlength{\tabcolsep}{4pt}

% --- Images constrained to page ---
\usepackage{grffile}
\usepackage[export]{adjustbox}
\let\oldincludegraphics\includegraphics
\renewcommand{\includegraphics}[2][]{%
  \oldincludegraphics[max width=\textwidth,max height=0.45\textheight,keepaspectratio,#1]{#2}%
}

% --- Figures don't float ---
\usepackage{float}
\floatplacement{figure}{H}

% --- Caption styling ---
\usepackage{caption}
\captionsetup{labelformat=empty,font={small,color=gray},skip=4pt}

% --- Tighter lists ---
\usepackage{enumitem}
\setlist{nosep,leftmargin=1.5em}

% --- Links ---
\usepackage{hyperref}
\hypersetup{
  colorlinks=true,
  linkcolor=linkcolor,
  urlcolor=linkcolor,
  citecolor=linkcolor
}

% --- Horizontal rules ---
\renewcommand{\rule}[2]{\textcolor{headrulecolor}{\vrule width \textwidth height 0.5pt}}
`)

	// --- Watermark ---
	if cfg.Watermark != "" {
		escaped := latexEscape(cfg.Watermark)
		b.WriteString(fmt.Sprintf(`
\usepackage{eso-pic}
\usepackage{tikz}
\AddToShipoutPictureFG{%%
  \begin{tikzpicture}[remember picture,overlay]
    \node[rotate=45,opacity=0.12,scale=10,text=red] at (current page.center) {\textsf{\textbf{\MakeUppercase{%s}}}};
  \end{tikzpicture}%%
}
`, escaped))
	}

	// --- Section numbering ---
	if cfg.IsNumbered() {
		b.WriteString("\\setcounter{secnumdepth}{4}\n")
		if cfg.NumberFrom >= 2 {
			b.WriteString(`\makeatletter
\renewcommand{\thesection}{}
\renewcommand{\thesubsection}{\arabic{subsection}}
\renewcommand{\thesubsubsection}{\thesubsection.\arabic{subsubsection}}
`)
			b.WriteString(fmt.Sprintf("\\titleformat{\\section}\n  {\\LARGE\\headingfont\\bfseries\\color{accentdark}\\addfontfeatures{LetterSpace=%d}%s}\n  {}{0em}{}[\\vspace{2pt}{\\color{headrulecolor}\\titlerule[1pt]}]\n", t.H1LetterSpace, scshape))
			b.WriteString("\\makeatother\n")
		}
		if cfg.NumberFrom >= 3 {
			b.WriteString(`\renewcommand{\thesubsection}{}
\renewcommand{\thesubsubsection}{\arabic{subsubsection}}
\titleformat{\subsection}
  {\Large\headingfont\bfseries\color{accentdark}\addfontfeatures{LetterSpace=-1}}
  {}{0em}{}
`)
		}
	}

	return b.String()
}

// GenerateTitleBanner returns LaTeX for the title page banner.
func GenerateTitleBanner(cfg *config.Config) string {
	if cfg.Title == "" {
		return "\\renewcommand{\\maketitle}{}\n"
	}

	var b strings.Builder
	b.WriteString("\\makeatletter\n\\renewcommand{\\maketitle}{%\n")
	b.WriteString("  \\thispagestyle{fancy}%\n")
	b.WriteString("  \\vspace*{-\\topskip}%\n")
	b.WriteString("  \\vspace*{-\\headsep}%\n")
	b.WriteString("  \\vspace*{-\\headheight}%\n")
	b.WriteString("  \\vspace*{-0.55in}%\n")
	b.WriteString("  \\noindent\\hspace*{-0.5in}%\n")
	b.WriteString("  \\fcolorbox{titlebg}{titlebg}{%\n")
	b.WriteString("    \\parbox{\\dimexpr\\paperwidth-2\\fboxsep-2\\fboxrule}{%\n")
	b.WriteString("      \\hspace*{0.3in}\\begin{minipage}{\\dimexpr\\textwidth}%\n")
	b.WriteString("        \\vspace{20pt}%\n")

	b.WriteString(fmt.Sprintf("        {\\fontsize{28}{34}\\selectfont\\bfseries\\color{black}%s}\\\\[6pt]%%\n", latexEscape(cfg.Title)))

	if cfg.Subtitle != "" {
		b.WriteString(fmt.Sprintf("        {\\fontsize{14}{18}\\selectfont\\color{black}%s}\\\\[8pt]%%\n", latexEscape(cfg.Subtitle)))
	}
	if cfg.Author != "" {
		b.WriteString(fmt.Sprintf("        {\\fontsize{11}{14}\\selectfont\\color{black}%s}\\\\[6pt]%%\n", latexEscape(cfg.Author)))
	}
	if cfg.Date != "" {
		b.WriteString(fmt.Sprintf("        {\\fontsize{10}{12}\\selectfont\\color{black}%s}\\\\[4pt]%%\n", latexEscape(cfg.Date)))
	}

	b.WriteString("        \\vspace{6pt}%\n")
	b.WriteString("      \\end{minipage}%\n")
	b.WriteString("    }%\n")
	b.WriteString("  }%\n")
	b.WriteString("  \\par\\vspace{20pt}%\n")
	b.WriteString("}\n\\makeatother\n")
	b.WriteString("\\AtBeginDocument{\\maketitle}\n")

	return b.String()
}

func writeColor(b *strings.Builder, name, hex string) {
	b.WriteString(fmt.Sprintf("\\definecolor{%s}{HTML}{%s}\n", name, hex))
}

// latexEscape escapes special LaTeX characters in user-provided text.
func latexEscape(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\textbackslash{}`,
		`&`, `\&`,
		`%`, `\%`,
		`$`, `\$`,
		`#`, `\#`,
		`_`, `\_`,
		`{`, `\{`,
		`}`, `\}`,
		`~`, `\textasciitilde{}`,
		`^`, `\textasciicircum{}`,
	)
	return replacer.Replace(s)
}
