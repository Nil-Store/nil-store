#!/usr/bin/env bash
set -euo pipefail

# Build whitepaper.md into a PDF using pandoc.
# Uses a Unicode-capable engine to avoid LaTeX Unicode errors.

SCRIPT_NAME=$(basename "$0")
ROOT_DIR=$(cd -- "$(dirname -- "$0")" >/dev/null 2>&1 && pwd -P)
INPUT_MD="${1:-whitepaper.md}"

if [[ ! -f "$INPUT_MD" ]];
then
  # If not found relative to CWD, try repo root
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

# Choose a PDF engine that supports Unicode.
# Preference: xelatex > lualatex > wkhtmltopdf
PDF_ENGINE=""
if command -v xelatex >/dev/null 2>&1; then
  PDF_ENGINE="xelatex"
elif command -v lualatex >/dev/null 2>&1; then
  PDF_ENGINE="lualatex"
elif command -v wkhtmltopdf >/dev/null 2>&1; then
  PDF_ENGINE="wkhtmltopdf"
fi

if [[ -z "$PDF_ENGINE" ]]; then
  echo "[$SCRIPT_NAME] Warning: No Unicode-capable LaTeX engine (xelatex/lualatex) or wkhtmltopdf found." >&2
  echo "[$SCRIPT_NAME] Attempting with pandoc default engine (may hit Unicode errors)." >&2
fi

BASENAME=$(basename "$INPUT_MD" .md)
OUT_PDF="${BASENAME}.pdf"

echo "[$SCRIPT_NAME] Building $OUT_PDF from $INPUT_MD ..."

# Fonts: use LaTeX defaults (Computer/Latin Modern) unless PDF_FONTS=auto
FONT_ARGS=()
if [[ "${PDF_FONTS:-default}" == "auto" ]]; then
  MAINFONT=""; MONOFONT=""; MATHFONT=""
  if command -v fc-list >/dev/null 2>&1; then
    have_font() { fc-list ":family=$1" | grep -q .; }
    if have_font "STIXGeneral"; then MAINFONT="STIXGeneral"; fi
    if have_font "STIX Two Text" && [[ -z "$MAINFONT" ]]; then MAINFONT="STIX Two Text"; fi
    if have_font "Noto Serif" && [[ -z "$MAINFONT" ]]; then MAINFONT="Noto Serif"; fi
    if have_font "Times New Roman" && [[ -z "$MAINFONT" ]]; then MAINFONT="Times New Roman"; fi
    if have_font "DejaVu Serif" && [[ -z "$MAINFONT" ]]; then MAINFONT="DejaVu Serif"; fi

    if have_font "Menlo"; then MONOFONT="Menlo"; fi
    if have_font "JetBrains Mono" && [[ -z "$MONOFONT" ]]; then MONOFONT="JetBrains Mono"; fi
    if have_font "DejaVu Sans Mono" && [[ -z "$MONOFONT" ]]; then MONOFONT="DejaVu Sans Mono"; fi
    if have_font "Courier New" && [[ -z "$MONOFONT" ]]; then MONOFONT="Courier New"; fi

    if have_font "Latin Modern Math"; then MATHFONT="Latin Modern Math"; fi
    if have_font "STIX Two Math" && [[ -z "$MATHFONT" ]]; then MATHFONT="STIX Two Math"; fi
    if have_font "XITS Math" && [[ -z "$MATHFONT" ]]; then MATHFONT="XITS Math"; fi
    if have_font "TeX Gyre Termes Math" && [[ -z "$MATHFONT" ]]; then MATHFONT="TeX Gyre Termes Math"; fi
  fi
  if [[ -n "$MAINFONT" ]]; then FONT_ARGS+=( -V "mainfont=$MAINFONT" ); fi
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

# Ensure title appears on its own page by forcing a page break
# right after \maketitle. We do this by redefining \maketitle
# to add a clearpage and suppress page number on the title page.
TITLE_HOOK_FILE=""
TITLE_HOOK_FILE=$(mktemp 2>/dev/null || mktemp -t pandoc_title_hook)
cat >"$TITLE_HOOK_FILE" <<'LATEX'
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
trap 'rm -f "$TITLE_HOOK_FILE"' EXIT

set -x
PANDOC_ARGS=( )
PANDOC_ARGS+=( "$INPUT_MD" )
if [[ -n "${PDF_ENGINE}" ]]; then PANDOC_ARGS+=( "--pdf-engine=$PDF_ENGINE" ); fi
PANDOC_ARGS+=( --from "markdown+smart+tex_math_dollars+tex_math_single_backslash+table_captions+link_attributes" )
PANDOC_ARGS+=( -V colorlinks=true -V urlcolor=blue -V linkcolor=blue -V toccolor=black )
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
