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

# Look for command file in log directory first, then fallback to script directory
CMD_LIST_FILE=""
for cmd_file in "$LOG_DIR"/*.cmd; do
    if [ -f "$cmd_file" ]; then
        CMD_LIST_FILE="$cmd_file"
        break
    fi
done

# Fallback to default cmd-list.cmd in script directory
if [ -z "$CMD_LIST_FILE" ] || [ ! -f "$CMD_LIST_FILE" ]; then
    CMD_LIST_FILE="$SCRIPT_DIR/cmd-list.cmd"
fi

if [ ! -f "$CMD_LIST_FILE" ]; then
    echo "Command file not found in log directory or script directory"
    exit 3
fi

echo "Using command file: $CMD_LIST_FILE"
REPORT_FILE="$LOG_DIR/index.html"

if [ ! -d "$LOG_DIR" ]; then
  echo "Log directory not found: $LOG_DIR"
  exit 2
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


# Generate single section for all commands
echo "<h2>CB-Spider API Memory Usage Analysis Results</h2>" >> "$REPORT_FILE"
echo '<table><tr><th>Analysis Item</th></tr>' >> "$REPORT_FILE"

# Process all commands in order from cmd file
COMMAND_COUNT=0
while IFS= read -r command || [ -n "$command" ]; do
    # Skip blank, whitespace, or comment lines (start with # or //)
    if [[ -z "${command}" || "${command}" =~ ^[[:space:]]*$ || "${command}" =~ ^[[:space:]]*# || "${command}" =~ ^[[:space:]]*// ]]; then
        continue
    fi
    
    COMMAND_COUNT=$((COMMAND_COUNT + 1))
    
    # Extract ConnectionName from command line
    CONN=""
    if [[ "${command}" =~ ConnectionName=([^[:space:]&]+) ]]; then
        CONN="${BASH_REMATCH[1]}"
    fi
    
    # Get all analysis files and sort them by timestamp extracted from filename
    # Extract timestamp from filename pattern: *_YYYYMMDD_HHMMSS.png
    ALL_GRAPH_FILES=($(ls "$LOG_DIR"/analysis_*.png 2>/dev/null | while read -r file; do
        # Extract timestamp from filename (last occurrence of pattern _YYYYMMDD_HHMMSS)
        timestamp=$(basename "$file" | grep -oE '_[0-9]{8}_[0-9]{6}' | tail -1 | sed 's/^_//')
        echo "$timestamp $file"
    done | sort | cut -d' ' -f2-))
    
    ALL_SUMMARY_FILES=($(ls "$LOG_DIR"/analysis_*_summary.txt 2>/dev/null | while read -r file; do
        # Extract timestamp from filename (last occurrence of pattern _YYYYMMDD_HHMMSS)
        timestamp=$(basename "$file" | grep -oE '_[0-9]{8}_[0-9]{6}' | tail -1 | sed 's/^_//')
        echo "$timestamp $file"
    done | sort | cut -d' ' -f2-))
    
    # Use the command count to select the appropriate file (array index starts from 0)
    GRAPH_FILE=""
    SUMMARY_FILE=""
    if [ ${#ALL_GRAPH_FILES[@]} -gt 0 ] && [ $((COMMAND_COUNT - 1)) -lt ${#ALL_GRAPH_FILES[@]} ]; then
        GRAPH_FILE="${ALL_GRAPH_FILES[$((COMMAND_COUNT - 1))]}"
    fi
    if [ ${#ALL_SUMMARY_FILES[@]} -gt 0 ] && [ $((COMMAND_COUNT - 1)) -lt ${#ALL_SUMMARY_FILES[@]} ]; then
        SUMMARY_FILE="${ALL_SUMMARY_FILES[$((COMMAND_COUNT - 1))]}"
    fi
    
    # Extract memory info from summary if available
    if [ -f "$SUMMARY_FILE" ]; then
        AVG=$(grep 'Average' "$SUMMARY_FILE" | grep -oE '[0-9.]+ MB' | head -1)
        PEAK=$(grep 'Peak' "$SUMMARY_FILE" | grep -oE '[0-9.]+ MB' | head -1)
        MEM_SUMMARY="Average: $AVG / Peak: $PEAK"
    else
        MEM_SUMMARY="-"
    fi
    
    # Insert row: single cell with command, graph, and link
    echo "<tr><td>" >> "$REPORT_FILE"
    
    # Extract output file name from command line (after '>')
    OUTFILE=$(echo "$command" | awk -F '>' '{if (NF>1) print $2}' | xargs)
    if [ -n "$OUTFILE" ] && [ -f "$LOG_DIR/$OUTFILE" ]; then
        OUTLINK="<a href='$OUTFILE' target='_blank' rel='noopener noreferrer' onclick=\"window.open(this.href,'_blank','noopener,noreferrer,width=1200,height=900');return false;\">$OUTFILE</a>"
    else
        OUTLINK="$OUTFILE"
    fi
    
    echo "<div><b>Command $COMMAND_COUNT:</b> <pre style='display:inline;'>$command</pre>" >> "$REPORT_FILE"
    if [ -n "$OUTLINK" ]; then
        echo " (Output: $OUTLINK)" >> "$REPORT_FILE"
    fi
    echo "</div>" >> "$REPORT_FILE"
    
    if [ -f "$GRAPH_FILE" ]; then
        IMG_BASENAME="$(basename "$GRAPH_FILE")"
        # Handle filenames with quotes by encoding them for HTML
        IMG_BASENAME_SAFE=$(echo "$IMG_BASENAME" | sed 's/"/%22/g' | sed "s/'/%27/g")
        echo "<div class='img-section'><a href='$IMG_BASENAME_SAFE' target='_blank' rel='noopener noreferrer' onclick=\"window.open(this.href,'_blank','noopener,noreferrer,width=1800,height=1400');return false;\"><img src='$IMG_BASENAME_SAFE' width='1200' style='cursor:zoom-in;'></a></div>" >> "$REPORT_FILE"
    fi
    
    echo "</td></tr>" >> "$REPORT_FILE"
    
done < "$CMD_LIST_FILE"

echo '</table>' >> "$REPORT_FILE"

echo "<hr><p style='font-size:0.9em;color:#888;'>This report was generated automatically.</p></body></html>" >> "$REPORT_FILE"

echo "Report generated: $REPORT_FILE"
