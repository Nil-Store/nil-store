#!/usr/bin/env bash
set -euo pipefail

# Build whitepaper.md into a PDF using pandoc with strong Unicode and typography defaults.

SCRIPT_NAME=$(basename "$0")
ROOT_DIR=$(cd -- "$(dirname -- "$0")" >/dev/null 2>&1 && pwd -P)
INPUT_MD="${1:-whitepaper.md}"

if [[ ! -f "$INPUT_MD" ]]; then
  if [[ -f "$ROOT_DIR/whitepaper.md" ]]; then
    INPUT_MD="$ROOT_DIR/whitepaper.md"
  else
    echo "[$SCRIPT_NAME] Error: input Markdown file not found: $INPUT_MD" >&2
    echo "Usage: $SCRIPT_NAME [path/to/whitepaper.md]" >&2
    exit 1
  fi
fi

if ! command -v pandoc >/dev/null 2>&1; then
  echo "[$SCRIPT_NAME] Error: pandoc is not installed or not in PATH" >&2
  exit 1
fi

# Choose a PDF engine that supports Unicode and microtype well.
# Preference: lualatex > xelatex > wkhtmltopdf
PDF_ENGINE=""
if command -v lualatex >/dev/null 2>&1; then
  PDF_ENGINE="lualatex"
elif command -v xelatex >/dev/null 2>&1; then
  PDF_ENGINE="xelatex"
elif command -v wkhtmltopdf >/dev/null 2>&1; then
  PDF_ENGINE="wkhtmltopdf"
fi

if [[ -z "$PDF_ENGINE" ]]; then
  echo "[$SCRIPT_NAME] Warning: No Unicode-capable LaTeX engine (lualatex/xelatex) or wkhtmltopdf found." >&2
  echo "[$SCRIPT_NAME] Attempting with pandoc default engine (may hit Unicode issues)." >&2
fi

BASENAME=$(basename "$INPUT_MD" .md)
OUT_PDF="${BASENAME}.pdf"

echo "[$SCRIPT_NAME] Building $OUT_PDF from $INPUT_MD ..."

# Fonts: default to a darker serif if available (can override with PDF_FONTS=default)
FONT_ARGS=()
FONT_MODE="${PDF_FONTS:-dark}"
if [[ "$FONT_MODE" != "default" ]]; then
  MAINFONT=""; MONOFONT=""; MATHFONT=""; SANSFONT=""; USE_SANS_HEADINGS=0
  if command -v fc-list >/dev/null 2>&1; then
    have_font() { fc-list ":family=$1" | grep -q .; }
    if [[ "$FONT_MODE" == "satoshi" ]]; then
      # Use Satoshi for headings (sans), keep body in dark serif
      if have_font "Satoshi"; then SANSFONT="Satoshi"; USE_SANS_HEADINGS=1; fi
      if [[ -z "$SANSFONT" ]] && have_font "Satoshi Variable"; then SANSFONT="Satoshi Variable"; USE_SANS_HEADINGS=1; fi
      # Math font to pair with serif body
      if have_font "STIX Two Math"; then MATHFONT="STIX Two Math"; fi
      if [[ -z "$MATHFONT" ]] && have_font "Latin Modern Math"; then MATHFONT="Latin Modern Math"; fi
      if [[ -z "$MATHFONT" ]] && have_font "TeX Gyre Termes Math"; then MATHFONT="TeX Gyre Termes Math"; fi
      # Always choose a serif body face via dark serif chain below
      FONT_MODE="dark"
    fi
    if [[ "$FONT_MODE" == "dark" ]]; then
      # Prefer darker/text-rich serif + matching math
      if [[ -z "$MAINFONT" ]] && have_font "Libertinus Serif"; then MAINFONT="Libertinus Serif"; fi
      if [[ -z "$MATHFONT" ]] && have_font "Libertinus Math"; then MATHFONT="Libertinus Math"; fi
      if [[ -z "$MAINFONT" ]] && have_font "TeX Gyre Termes"; then MAINFONT="TeX Gyre Termes"; fi
      if [[ -z "$MATHFONT" ]] && have_font "TeX Gyre Termes Math"; then MATHFONT="TeX Gyre Termes Math"; fi
      if [[ -z "$MAINFONT" ]] && have_font "TeX Gyre Pagella"; then MAINFONT="TeX Gyre Pagella"; fi
      if [[ -z "$MATHFONT" ]] && have_font "TeX Gyre Pagella Math"; then MATHFONT="TeX Gyre Pagella Math"; fi
      if [[ -z "$MAINFONT" ]] && have_font "STIX Two Text"; then MAINFONT="STIX Two Text"; fi
      if [[ -z "$MATHFONT" ]] && have_font "STIX Two Math"; then MATHFONT="STIX Two Math"; fi
      if [[ -z "$MAINFONT" ]] && have_font "STIXGeneral"; then MAINFONT="STIXGeneral"; fi
    fi
    # Monospace preference with broader glyphs
    if have_font "Noto Sans Mono"; then MONOFONT="Noto Sans Mono"; fi
    if [[ -z "$MONOFONT" ]] && have_font "Menlo"; then MONOFONT="Menlo"; fi
  fi
  if [[ -n "$MAINFONT" ]]; then FONT_ARGS+=( -V "mainfont=$MAINFONT" ); fi
  if [[ -n "$SANSFONT" ]]; then FONT_ARGS+=( -V "sansfont=$SANSFONT" ); fi
  if [[ -n "$MONOFONT" ]]; then FONT_ARGS+=( -V "monofont=$MONOFONT" ); fi
  if [[ -n "$MATHFONT" ]]; then FONT_ARGS+=( -V "mathfont=$MATHFONT" ); fi
fi

# Ensure the title page is set so that the TOC appears after the title.
# Extract first H1 as title; fallback to file base name.
TITLE_VAL=$(awk '/^# /{sub(/^# /,""); print; exit}' "$INPUT_MD")
if [[ -z "$TITLE_VAL" ]]; then TITLE_VAL="$BASENAME"; fi
META_ARGS=( --metadata=title="$TITLE_VAL" )

# Page geometry (reduce margins / padding). Override with env PDF_MARGIN, e.g. PDF_MARGIN=0.7in
PDF_MARGIN_DEFAULT="0.8in"
PDF_MARGIN="${PDF_MARGIN:-$PDF_MARGIN_DEFAULT}"
GEOMETRY_ARGS=( -V "geometry:margin=$PDF_MARGIN" -V "geometry:heightrounded" )

# Font size across the document. Override with env PDF_FONTSIZE (e.g., 11pt, 12pt)
FONTSIZE_DEFAULT="12pt"
FONTSIZE="${PDF_FONTSIZE:-$FONTSIZE_DEFAULT}"
CLASSOPT_ARGS=( -V "classoption=$FONTSIZE" )

# Build a header snippet (LaTeX) for title/TOC handling and micro-typography.
TITLE_HOOK_FILE=$(mktemp 2>/dev/null || mktemp -t pandoc_title_hook)
cat >"$TITLE_HOOK_FILE" <<'LATEX'
% Typography improvements and title/TOC formatting
% Engine-aware font encoding handling and micro-typography
\usepackage{iftex}
\ifPDFTeX
  \usepackage[T1]{fontenc}
  \usepackage{lmodern}
\fi
% Microtype basic math protrusion if the template already loaded microtype
\makeatletter
\@ifpackageloaded{microtype}{\UseMicrotypeSet[protrusion]{basicmath}}{}
\makeatother

% Map common Unicode math/Greek symbols in text to math mode for full glyph coverage
\usepackage{newunicodechar}
\newunicodechar{≥}{\ensuremath{\ge}}
\newunicodechar{≤}{\ensuremath{\le}}
\newunicodechar{≈}{\ensuremath{\approx}}
\newunicodechar{→}{\ensuremath{\to}}
\newunicodechar{∥}{\ensuremath{\parallel}}
\newunicodechar{‖}{\ensuremath{\Vert}}
\newunicodechar{∝}{\ensuremath{\propto}}
\newunicodechar{σ}{\ensuremath{\sigma}}
\newunicodechar{θ}{\ensuremath{\theta}}
\newunicodechar{λ}{\ensuremath{\lambda}}
\newunicodechar{τ}{\ensuremath{\tau}}
\newunicodechar{ε}{\ensuremath{\varepsilon}}
\newunicodechar{π}{\ensuremath{\pi}}
\newunicodechar{δ}{\ensuremath{\delta}}
% Non-breaking hyphen fallback
\newunicodechar{‑}{-}
% Square root symbol as a standalone surd
\newunicodechar{√}{\ensuremath{\surd}}

% Centered (vertically) title page and page breaks
\makeatletter
\AtBeginDocument{%
  % Custom centered title page without page number
  \let\origmaketitle\maketitle
  \renewcommand{\maketitle}{%
    \begin{titlepage}
      \thispagestyle{empty}
      \begin{center}
        \vspace*{\fill}
        {\LARGE\bfseries \@title\par}\vspace{1em}% title
        \ifx\@author\@empty\else {\large \@author\par}\vspace{0.5em}\fi
        \ifx\@date\@empty\else {\large \@date\par}\fi
        \vspace*{\fill}
      \end{center}
    \end{titlepage}% end title page
  }%

  % Insert a page break right after the Table of Contents
  \let\origtableofcontents\tableofcontents
  \renewcommand{\tableofcontents}{\origtableofcontents\clearpage}
}
\makeatother
LATEX
# Optional: add right padding for lists using enumitem
LIST_RIGHT_MARGIN="${PDF_LIST_RIGHTMARGIN:-1em}"
{
  echo "\\usepackage{enumitem}";
  echo "\\setlist[itemize]{rightmargin=${LIST_RIGHT_MARGIN}}";
  echo "\\setlist[enumerate]{rightmargin=${LIST_RIGHT_MARGIN}}";
  echo "\\setlist[description]{rightmargin=${LIST_RIGHT_MARGIN}}";
} >>"$TITLE_HOOK_FILE"
if [[ ${USE_SANS_HEADINGS:-0} -eq 1 ]]; then
  {
    echo "\\usepackage{sectsty}";
    # Choose which heading levels use sans (Satoshi) for a strong hierarchy.
    # Default: H1 and H2 only. Override with env PDF_SANS_HEADINGS (e.g., 1,3 or 1,2,3).
    echo "% Satoshi for selected heading levels";
  } >>"$TITLE_HOOK_FILE"
  LEVELS_CSV=${PDF_SANS_HEADINGS:-1,2}
  IFS=',' read -r -a LEVELS_ARR <<< "$LEVELS_CSV"
  for lvl in "${LEVELS_ARR[@]}"; do
    case "$lvl" in
      1) echo "\\sectionfont{\\sffamily\\bfseries}" >>"$TITLE_HOOK_FILE";;
      2) echo "\\subsectionfont{\\sffamily\\bfseries}" >>"$TITLE_HOOK_FILE";;
      3) echo "\\subsubsectionfont{\\sffamily\\bfseries}" >>"$TITLE_HOOK_FILE";;
      4) echo "\\paragraphfont{\\sffamily\\bfseries}" >>"$TITLE_HOOK_FILE";;
      5) echo "\\subparagraphfont{\\sffamily\\bfseries}" >>"$TITLE_HOOK_FILE";;
      *) :;;
    esac
  done
fi
trap 'rm -f "$TITLE_HOOK_FILE"' EXIT

set -x
PANDOC_ARGS=( )
PANDOC_ARGS+=( "$INPUT_MD" )
if [[ -n "${PDF_ENGINE}" ]]; then PANDOC_ARGS+=( "--pdf-engine=$PDF_ENGINE" ); fi
PANDOC_ARGS+=( --from "markdown+smart+tex_math_dollars+tex_math_single_backslash+table_captions+link_attributes" )
PANDOC_ARGS+=( -V colorlinks=true -V urlcolor=blue -V linkcolor=blue -V toccolor=black )
# Enable microtype via pandoc template (avoids option clashes)
PANDOC_ARGS+=( -V microtype=true -V microtypeoptions=protrusion=true -V microtypeoptions=expansion=true )
if [[ ${#FONT_ARGS[@]} -gt 0 ]]; then PANDOC_ARGS+=( "${FONT_ARGS[@]}" ); fi
PANDOC_ARGS+=( "${META_ARGS[@]}" )
PANDOC_ARGS+=( --include-in-header "$TITLE_HOOK_FILE" )
PANDOC_ARGS+=( "${GEOMETRY_ARGS[@]}" )
PANDOC_ARGS+=( "${CLASSOPT_ARGS[@]}" )
PANDOC_ARGS+=( --toc )
PANDOC_ARGS+=( -o "$OUT_PDF" )

pandoc "${PANDOC_ARGS[@]}"
set +x

echo "[$SCRIPT_NAME] Done: $OUT_PDF"
