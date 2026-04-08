// script.go generates the bash conversion script that runs inside the Docker container.
// This script handles: callout conversion, page break injection, mermaid rendering,
// frontmatter stripping, and pandoc invocation. It's written to a temp file and
// mounted into the container.
package converter

import (
	"fmt"
	"strings"

	"github.com/jclement/pdfify/internal/config"
	"github.com/jclement/pdfify/internal/theme"
)

// GenerateScript creates the bash conversion script that runs inside Docker.
// The script performs all markdown preprocessing and invokes pandoc with XeLaTeX.
func GenerateScript(cfg *config.Config, t *theme.Theme) string {
	preamble := GeneratePreamble(cfg, t)
	titleBanner := GenerateTitleBanner(cfg)
	fullPreamble := preamble + "\n" + titleBanner

	// Escape the preamble for embedding in a heredoc
	escapedPreamble := strings.ReplaceAll(fullPreamble, "'", "'\\''")

	mermaidConfig := t.MermaidConfigJSON()

	var b strings.Builder
	b.WriteString("#!/bin/bash\n")
	b.WriteString("set -euo pipefail\n\n")

	b.WriteString("INPUT_FILE=\"$1\"\n")
	b.WriteString("OUTPUT_FILE=\"$2\"\n")
	b.WriteString("WORKDIR=\"/work\"\n")
	b.WriteString("FENCE=$'\\x60\\x60\\x60'\n")
	b.WriteString("cd \"$WORKDIR\"\n\n")

	// Write mermaid config
	b.WriteString(fmt.Sprintf("echo '%s' > /tmp/mermaid-config.json\n", mermaidConfig))
	b.WriteString("echo '{\"args\": [\"--no-sandbox\", \"--disable-setuid-sandbox\", \"--disable-dev-shm-usage\", \"--disable-gpu\"]}' > /tmp/puppeteer-config.json\n\n")

	// Strip first H1 if needed
	b.WriteString(fmt.Sprintf("HIDE_FIRST_H1=\"${HIDE_FIRST_H1:-0}\"\n"))
	b.WriteString(`EFFECTIVE_INPUT="$INPUT_FILE"
if [[ "$HIDE_FIRST_H1" == "1" ]]; then
    STRIPPED=$(mktemp /tmp/pdfify-stripped-XXXXXX.md)
    FOUND_H1=0
    IN_CODE_BLK=0
    IN_FMATTER=0
    while IFS= read -r line || [[ -n "$line" ]]; do
        [[ "$line" =~ ^$FENCE ]] && { if [[ $IN_CODE_BLK -eq 0 ]]; then IN_CODE_BLK=1; else IN_CODE_BLK=0; fi; }
        if [[ "$line" == "---" && $IN_CODE_BLK -eq 0 ]]; then
            if [[ $IN_FMATTER -eq 0 && $FOUND_H1 -eq 0 ]]; then IN_FMATTER=1; else IN_FMATTER=0; fi
        fi
        if [[ $FOUND_H1 -eq 0 && $IN_CODE_BLK -eq 0 && $IN_FMATTER -eq 0 && "$line" =~ ^#\  ]]; then
            FOUND_H1=1; continue
        fi
        if [[ $FOUND_H1 -eq 1 && -z "$line" ]]; then FOUND_H1=2; continue; fi
        [[ $FOUND_H1 -eq 1 ]] && FOUND_H1=2
        echo "$line" >> "$STRIPPED"
    done < "$INPUT_FILE"
    EFFECTIVE_INPUT="$(basename "$STRIPPED")"
fi

`)

	// Callout processing
	b.WriteString(`# --- Callout processing ---
CALLOUT_MD=$(mktemp /tmp/pdfify-callout-XXXXXX.md)
IN_CALLOUT=0; CALLOUT_TYPE=""; CALLOUT_TITLE=""; CALLOUT_BUF=""

flush_callout() {
    if [[ $IN_CALLOUT -eq 1 && -n "$CALLOUT_TYPE" ]]; then
        local lt
        case "${CALLOUT_TYPE,,}" in
            info|note) lt="calloutinfo" ;; tip|hint) lt="callouttip" ;;
            warning|caution) lt="calloutwarning" ;; danger|error|bug) lt="calloutdanger" ;;
            example) lt="calloutexample" ;; quote|cite) lt="calloutquote" ;; *) lt="calloutinfo" ;;
        esac
        echo "" >> "$CALLOUT_MD"
        printf '%s\n' '` + "```{=latex}" + `' >> "$CALLOUT_MD"
        echo "\\begin{${lt}}{${CALLOUT_TITLE}}" >> "$CALLOUT_MD"
        printf '%s\n' '` + "```" + `' >> "$CALLOUT_MD"
        echo "" >> "$CALLOUT_MD"
        echo "$CALLOUT_BUF" >> "$CALLOUT_MD"
        echo "" >> "$CALLOUT_MD"
        printf '%s\n' '` + "```{=latex}" + `' >> "$CALLOUT_MD"
        echo "\\end{${lt}}" >> "$CALLOUT_MD"
        printf '%s\n' '` + "```" + `' >> "$CALLOUT_MD"
        echo "" >> "$CALLOUT_MD"
    fi
    IN_CALLOUT=0; CALLOUT_TYPE=""; CALLOUT_TITLE=""; CALLOUT_BUF=""
}

while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" =~ ^\>\ *\[!([a-zA-Z]+)\]\ *(.*) ]]; then
        flush_callout; IN_CALLOUT=1; CALLOUT_TYPE="${BASH_REMATCH[1]}"
        CALLOUT_TITLE="${BASH_REMATCH[2]:-${BASH_REMATCH[1]^}}"; continue
    fi
    if [[ $IN_CALLOUT -eq 1 ]]; then
        if [[ "$line" =~ ^\>\ ?(.*) ]]; then
            CALLOUT_BUF="${CALLOUT_BUF}${BASH_REMATCH[1]}
"; continue
        else flush_callout; fi
    fi
    echo "$line" >> "$CALLOUT_MD"
done < "${STRIPPED:-$EFFECTIVE_INPUT}"
flush_callout

`)

	// Page break injection
	b.WriteString(fmt.Sprintf(`# --- Page breaks ---
BREAK_MD=$(mktemp /tmp/pdfify-breaks-XXXXXX.md)
H1_COUNT=0; IN_FM=0; IN_CODE=0; DONE_TOC_BREAK=0
TOC_LEVEL="%d"
FILE_NUMBER_FROM="%d"
FILE_PAGEBREAK="%d"

while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" =~ ^$FENCE ]]; then
        if [[ $IN_CODE -eq 0 ]]; then IN_CODE=1; else IN_CODE=0; fi
        echo "$line" >> "$BREAK_MD"; continue
    fi
    if [[ "$line" == "---" && $IN_CODE -eq 0 ]]; then
        if [[ $IN_FM -eq 0 && $H1_COUNT -eq 0 ]]; then IN_FM=1; else IN_FM=0; fi
        echo "$line" >> "$BREAK_MD"; continue
    fi
    if [[ $IN_CODE -eq 0 && $IN_FM -eq 0 ]]; then
        if [[ $DONE_TOC_BREAK -eq 0 && "$TOC_LEVEL" -gt 0 && -n "$line" ]]; then
            echo "" >> "$BREAK_MD"
            printf '%%s\n' '`+"```{=latex}"+`' >> "$BREAK_MD"
            echo '\newpage' >> "$BREAK_MD"
            printf '%%s\n' '`+"```"+`' >> "$BREAK_MD"
            echo "" >> "$BREAK_MD"
            DONE_TOC_BREAK=1
        fi
        BREAK_HASHES=$(printf '#%%.0s' $(seq 1 "$FILE_NUMBER_FROM"))
        if [[ "$line" == "${BREAK_HASHES} "* ]]; then
            NEXT_CHAR="${line:${#BREAK_HASHES}:1}"
            if [[ "$NEXT_CHAR" != "#" ]]; then
                H1_COUNT=$((H1_COUNT + 1))
                if [[ $H1_COUNT -gt 1 && $FILE_PAGEBREAK -eq 1 ]]; then
                    echo "" >> "$BREAK_MD"
                    printf '%%s\n' '`+"```{=latex}"+`' >> "$BREAK_MD"
                    echo '\newpage' >> "$BREAK_MD"
                    printf '%%s\n' '`+"```"+`' >> "$BREAK_MD"
                    echo "" >> "$BREAK_MD"
                fi
            fi
        fi
    fi
    echo "$line" >> "$BREAK_MD"
done < "$CALLOUT_MD"
rm -f "$CALLOUT_MD"

`, cfg.TocLevel, cfg.NumberFrom, boolToInt(cfg.IsPageBreak())))

	// Mermaid rendering
	b.WriteString(`# --- Mermaid rendering ---
TEMP_MD=$(mktemp /tmp/pdfify-XXXXXX.md)
MERMAID_COUNT=0; IN_MERMAID=0; MERMAID_BUF=""

while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" =~ ^${FENCE}mermaid ]]; then
        IN_MERMAID=1; MERMAID_BUF=""; continue
    fi
    if [[ $IN_MERMAID -eq 1 ]]; then
        if [[ "$line" =~ ^$FENCE ]]; then
            IN_MERMAID=0; MERMAID_COUNT=$((MERMAID_COUNT + 1))
            echo "$MERMAID_BUF" > "/tmp/mermaid-${MERMAID_COUNT}.mmd"
            mmdc -i "/tmp/mermaid-${MERMAID_COUNT}.mmd" -o "/tmp/mermaid-${MERMAID_COUNT}.png" \
                 -w 1600 -s 3 -b transparent -c /tmp/mermaid-config.json -p /tmp/puppeteer-config.json 2>/dev/null || {
                echo '` + "```" + `' >> "$TEMP_MD"
                echo "$MERMAID_BUF" >> "$TEMP_MD"
                echo '` + "```" + `' >> "$TEMP_MD"
                continue
            }
            echo "" >> "$TEMP_MD"
            echo "![Diagram ${MERMAID_COUNT}](/tmp/mermaid-${MERMAID_COUNT}.png)\\" >> "$TEMP_MD"
            echo "" >> "$TEMP_MD"
        else
            MERMAID_BUF="${MERMAID_BUF}${line}
"
        fi
    else
        echo "$line" >> "$TEMP_MD"
    fi
done < "$BREAK_MD"
rm -f "$BREAK_MD"

# Strip frontmatter from processed markdown
if head -1 "$TEMP_MD" | grep -q '^---'; then
    STRIPPED_FM=$(mktemp /tmp/pdfify-nofm-XXXXXX.md)
    awk 'NR==1 && /^---/{skip=1; next} skip && /^---/{skip=0; next} !skip' "$TEMP_MD" > "$STRIPPED_FM"
    mv "$STRIPPED_FM" "$TEMP_MD"
fi

`)

	// Lua filter for bracket protection in headings
	b.WriteString(`# --- Lua filter for bracket protection ---
cat > /tmp/bracket-filter.lua <<'LUAFILTER'
function Header(el)
  if FORMAT ~= "latex" and FORMAT ~= "pdf" then return nil end
  el = el:walk {
    Str = function(s)
      if s.text:find("[%[%]]") then
        local plain = s.text
        local t = s.text:gsub("%[", "{[}"):gsub("%]", "{]}")
        return pandoc.RawInline("latex", "\\texorpdfstring{" .. t .. "}{" .. plain .. "}")
      end
    end,
    Code = function(c)
      local plain = c.text
      local t = c.text
      t = t:gsub("\\", "\\textbackslash ")
      t = t:gsub("%%", "\\%%")
      t = t:gsub("%#", "\\#")
      t = t:gsub("%$", "\\$")
      t = t:gsub("%&", "\\&")
      t = t:gsub("_", "\\_")
      t = t:gsub("%{", "\\{")
      t = t:gsub("%}", "\\}")
      t = t:gsub("~", "\\textasciitilde{}")
      t = t:gsub("%^", "\\textasciicircum{}")
      t = t:gsub("%[", "{[}"):gsub("%]", "{]}")
      return pandoc.RawInline("latex", "\\texorpdfstring{\\oldtexttt{" .. t .. "}}{" .. plain .. "}")
    end
  }
  return el
end
LUAFILTER

`)

	// Write preamble to file
	b.WriteString("cat > /tmp/preamble.tex <<'PREAMBLE_EOF'\n")
	b.WriteString(fullPreamble)
	b.WriteString("\nPREAMBLE_EOF\n\n")

	// Pandoc invocation
	geometry := cfg.GeometryString()
	tocFlags := ""
	if cfg.TocLevel > 0 {
		tocFlags = fmt.Sprintf("--toc --toc-depth=%d", cfg.TocLevel)
	}
	numberFlags := ""
	if cfg.IsNumbered() {
		numberFlags = "--number-sections"
	}

	b.WriteString(fmt.Sprintf(`# --- Pandoc ---
pandoc "$TEMP_MD" \
    -o "$OUTPUT_FILE" \
    --pdf-engine=xelatex \
    --lua-filter=/tmp/bracket-filter.lua \
    --resource-path=".:$WORKDIR" \
    --columns=72 \
    -V geometry:"%s" \
    -V fontsize=10pt \
    -V mainfont="%s" \
    -V monofont="%s" \
    %s \
    %s \
    --highlight-style=tango \
    -H /tmp/preamble.tex \
    --standalone

rm -f "$TEMP_MD" /tmp/mermaid-*.mmd /tmp/mermaid-*.png /tmp/preamble.tex /tmp/bracket-filter.lua
`, geometry, t.MainFont, t.MonoFont, tocFlags, numberFlags))

	_ = escapedPreamble // used in heredoc above

	return b.String()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
