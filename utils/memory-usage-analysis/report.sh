#!/bin/bash
# report.sh: Generate HTML report for CB-Spider memory usage analysis
# Usage: ./report.sh <log-dir>

set -e

if [ $# -ne 1 ]; then
  echo "Usage: $0 <log-dir>"
  exit 1
fi

LOG_DIR="$1"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CMD_LIST_FILE="$SCRIPT_DIR/cmd-list.cmd"
if [ ! -f "$CMD_LIST_FILE" ]; then
  echo "cmd-list.cmd not found in $SCRIPT_DIR"
  exit 3
fi
REPORT_FILE="$LOG_DIR/index.html"

if [ ! -d "$LOG_DIR" ]; then
  echo "Log directory not found: $LOG_DIR"
  exit 2
fi
if [ ! -f "$CMD_LIST_FILE" ]; then
  echo "cmd-list.cmd not found in $LOG_DIR"
  exit 3
fi

# Extract analysis date from log dir name (format: log-YYYYMMDD_HHMMSS)
ANALYSIS_DATE=$(echo "$LOG_DIR" | sed -E 's/.*log-([0-9_]+).*/\1/' | sed 's/_/ /')

# Read Spider version info
SPIDER_VERSION_FILE="$LOG_DIR/spider_version.txt"
SPIDER_VERSION=""
if [ -f "$SPIDER_VERSION_FILE" ]; then
  # Extract version and commit info from the file
  VER_LINE=$(grep '^Version:' "$SPIDER_VERSION_FILE" | head -1 | awk '{print $2}')
  SHA_LINE=$(grep '^Commit SHA:' "$SPIDER_VERSION_FILE" | head -1 | awk '{print $3}')
  # Remove leading 'v' if present
  if [[ "$VER_LINE" == v* ]]; then
    VER_LINE="${VER_LINE#v}"
  fi
  if [ -n "$VER_LINE" ] && [ -n "$SHA_LINE" ]; then
    SPIDER_VERSION="v$VER_LINE-$SHA_LINE"
  fi
fi
if [ -z "$SPIDER_VERSION" ]; then
  SPIDER_VERSION="(unknown)"
fi

# Prepare HTML header (English)
cat > "$REPORT_FILE" <<EOF
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>CB-Spider $SPIDER_VERSION Memory Usage Analysis by API</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 30px; }
    h1, h2, h3 { color: #2E86AB; }
    table { border-collapse: collapse; width: 100%; margin-bottom: 30px; }
    th, td { border: 1px solid #aaa; padding: 8px; text-align: left; }
    th { background: #eaf6fb; }
    tr:nth-child(even) { background: #f7fbfd; }
    .img-section { margin: 20px 0; }
    .file-links { font-size: 0.95em; }
  </style>
</head>
<body>
<h1>CB-Spider $SPIDER_VERSION Memory Usage Analysis by API</h1>
<p><b>Analysis Date:</b> $ANALYSIS_DATE</p>
EOF


# 2. 장별 리포트 함수
generate_section() {
  local SECTION_TITLE="$1"
  local CMD_KEYWORD="$2"
  echo "<h2>$SECTION_TITLE</h2>" >> "$REPORT_FILE"
  echo '<table><tr><th>Analysis Item</th></tr>' >> "$REPORT_FILE"
  # Find matching commands in cmd-list.cmd
  grep -n "$CMD_KEYWORD" "$CMD_LIST_FILE" | while IFS=: read -r LINENO CMDLINE; do
    # Extract ConnectionName from command line
    CONN=$(echo "$CMDLINE" | grep -oE 'ConnectionName=[^ "\}]+' | head -1 | cut -d= -f2)
    # Compose a robust pattern for matching files
    if [[ "$CMD_KEYWORD" == "vmimage" ]]; then
      BASEPATTERN="analysis_curl_-sX_GET_http___localhost_1024_spider_vmimage__ConnectionName_${CONN}_*.png"
      SUMMARYPATTERN="analysis_curl_-sX_GET_http___localhost_1024_spider_vmimage__ConnectionName_${CONN}_*_summary.txt"
    else
      BASEPATTERN="analysis_curl_-sX_POST_http___localhost_1024_spider_pricein_ConnectionName_${CONN}_*.png"
      SUMMARYPATTERN="analysis_curl_-sX_POST_http___localhost_1024_spider_pricein_ConnectionName_${CONN}_*_summary.txt"
    fi
    GRAPH_FILE=$(ls "$LOG_DIR"/$BASEPATTERN 2>/dev/null | head -1)
    SUMMARY_FILE=$(ls "$LOG_DIR"/$SUMMARYPATTERN 2>/dev/null | head -1)
    # Extract memory info
    if [ -f "$SUMMARY_FILE" ]; then
      AVG=$(grep '평균' "$SUMMARY_FILE" | grep -oE '[0-9.]+ MB' | head -1)
      PEAK=$(grep 'Peak' "$SUMMARY_FILE" | grep -oE '[0-9.]+ MB' | head -1)
      MEM_SUMMARY="평균: $AVG / Peak: $PEAK"
    else
      MEM_SUMMARY="-"
    fi
    # Related files
    FILE_LINKS=""
    [ -f "$SUMMARY_FILE" ] && FILE_LINKS+="<a href='$(basename "$SUMMARY_FILE")'>요약</a> "
    [ -f "$GRAPH_FILE" ] && FILE_LINKS+="<a href='$(basename "$GRAPH_FILE")'>그래프</a> "
    # Insert row: single cell with command, graph, and link
    echo "<tr><td>" >> "$REPORT_FILE"
    # Extract output file name from command line (after '>')
    OUTFILE=$(echo "$CMDLINE" | awk -F '>' '{if (NF>1) print $2}' | xargs)
    if [ -n "$OUTFILE" ] && [ -f "$LOG_DIR/$OUTFILE" ]; then
      OUTLINK="<a href='$OUTFILE' target='_blank' rel='noopener noreferrer' onclick=\"window.open(this.href,'_blank','noopener,noreferrer,width=1200,height=900');return false;\">$OUTFILE</a>"
    else
      OUTLINK="$OUTFILE"
    fi
    echo "<div><b>Command:</b> <pre style='display:inline;'>$CMDLINE</pre>" >> "$REPORT_FILE"
    if [ -n "$OUTLINK" ]; then
      echo " (Output: $OUTLINK)" >> "$REPORT_FILE"
    fi
    echo "</div>" >> "$REPORT_FILE"
    if [ -f "$GRAPH_FILE" ]; then
      IMG_BASENAME="$(basename "$GRAPH_FILE")"
  echo "<div class='img-section'><a href='$IMG_BASENAME' target='_blank' rel='noopener noreferrer' onclick=\"window.open(this.href,'_blank','noopener,noreferrer,width=1800,height=1400');return false;\"><img src='$IMG_BASENAME' width='1200' style='cursor:zoom-in;'></a></div>" >> "$REPORT_FILE"
    fi
    echo "</td></tr>" >> "$REPORT_FILE"
  done
  echo '</table>' >> "$REPORT_FILE"
}

generate_section "1. Spider Server Memory Usage for Each CSP VM Image Info List API" "vmimage"
generate_section "2. Spider Server Memory Usage for Each CSP VM Price Info List API" "priceinfo"

echo "<hr><p style='font-size:0.9em;color:#888;'>This report was generated automatically.</p></body></html>" >> "$REPORT_FILE"

echo "Report generated: $REPORT_FILE"
