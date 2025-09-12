#!/bin/bash

# CB-Spider Server Memory Usage Auto Analysis Script
# Author: Auto Memory Analysis Script
# Created: $(date '+%Y-%m-%d %H:%M:%S')

set -e

 # Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

 # Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_header() {
    echo -e "${PURPLE}===================================================${NC}"
    echo -e "${PURPLE}$1${NC}"
    echo -e "${PURPLE}===================================================${NC}"
}

 # Script start
log_header "CB-Spider Server Memory Usage Analysis Started"
START_TIME=$(date '+%Y%m%d_%H%M%S')

# Check current directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}" )" && pwd)"
LOG_DIR="${SCRIPT_DIR}/log-${START_TIME}"

log_info "Script Directory: ${SCRIPT_DIR}"
log_info "Log Directory: ${LOG_DIR}"

# Create log directory (do this before version file)
mkdir -p "${LOG_DIR}"
# Save CB-Spider version info
SPIDER_VERSION_FILE="${LOG_DIR}/spider_version.txt"
if [ -x "${SCRIPT_DIR}/../../bin/cb-spider" ]; then
    "${SCRIPT_DIR}/../../bin/cb-spider" -v > "${SPIDER_VERSION_FILE}" 2>&1 || true
elif [ -x "${SCRIPT_DIR}/../../bin/spider" ]; then
    "${SCRIPT_DIR}/../../bin/spider" -v > "${SPIDER_VERSION_FILE}" 2>&1 || true
fi
log_info "Spider version info saved to: ${SPIDER_VERSION_FILE}"
# Read CB-Spider PID (bin directory two levels up from script)
PID_FILE="${SCRIPT_DIR}/../../bin/spider.pid"
if [ ! -f "${PID_FILE}" ]; then
    log_error "PID file not found: ${PID_FILE}"
    log_error "Please make sure CB-Spider server is running"
    exit 1
fi

SPIDER_PID=$(cat "${PID_FILE}")
log_info "CB-Spider PID: ${SPIDER_PID}"

# Check if PID is actually running
if ! ps -p "${SPIDER_PID}" > /dev/null 2>&1; then
    log_error "CB-Spider process (PID: ${SPIDER_PID}) is not running"
    exit 1
fi

log_success "CB-Spider server is running (PID: ${SPIDER_PID})"

# Command file check - use argument if present, else use env or default
if [ $# -gt 0 ]; then
    # Use first argument as command file
    if [[ "$1" == /* ]]; then
        # Absolute path
        CMD_FILE="$1"
    else
        # Relative path from script dir
        CMD_FILE="${SCRIPT_DIR}/$1"
    fi
else
    # No argument: use env or default
    CMD_FILE="${CMD_FILE:-${SCRIPT_DIR}/cmd-list.cmd}"
fi

if [ ! -f "${CMD_FILE}" ]; then
    log_error "Command file not found: ${CMD_FILE}"
    exit 1
fi

log_info "Command file: ${CMD_FILE}"

# Count commands (ignore # or // comments and blank lines)
TOTAL_COMMANDS=$(grep -v -E "^[[:space:]]*#|^[[:space:]]*//|^[[:space:]]*$" "${CMD_FILE}" | wc -l)
log_info "Total commands to execute: ${TOTAL_COMMANDS}"

# Create log directory
mkdir -p "${LOG_DIR}"

# Execute and analyze commands
COMMAND_COUNT=0
while IFS= read -r command || [ -n "$command" ]; do
    # Skip blank, whitespace, or comment lines (start with # or //)
    if [[ -z "${command}" || "${command}" =~ ^[[:space:]]*$ || "${command}" =~ ^[[:space:]]*# || "${command}" =~ ^[[:space:]]*// ]]; then
        continue
    fi
    
    COMMAND_COUNT=$((COMMAND_COUNT + 1))
    CMD_START_TIME=$(date '+%Y%m%d_%H%M%S')
    

    log_header "Command ${COMMAND_COUNT}/${TOTAL_COMMANDS}: Memory Analysis"
    log_info "Command: ${command}"
    
    # Extract ConnectionName (from URL param or JSON data)
    CONNECTION_NAME=""
    if [[ "${command}" =~ ConnectionName=([^[:space:]&]+) ]]; then
        # Extract from URL param
        CONNECTION_NAME="${BASH_REMATCH[1]}"
    elif [[ "${command}" =~ \"ConnectionName\"[[:space:]]*:[[:space:]]*\"([^\"]+)\" ]]; then
        # Extract from JSON data
        CONNECTION_NAME="${BASH_REMATCH[1]}"
    fi
    
    # Create filename (remove special chars, include ConnectionName)
    CMD_SAFE=$(echo "${command}" | sed 's/[^a-zA-Z0-9._-]/_/g' | cut -c1-50)
    if [[ -n "${CONNECTION_NAME}" ]]; then
        CMD_SAFE="${CMD_SAFE}_ConnectionName_${CONNECTION_NAME}"
    fi
    BASE_FILENAME="${CMD_SAFE}_${CMD_START_TIME}"
    
    LOG_FILE="${LOG_DIR}/${BASE_FILENAME}.log"
    SYSTEM_MEMORY_FILE="${LOG_DIR}/system_memory_${BASE_FILENAME}.log"
    ANALYSIS_PREFIX="${LOG_DIR}/analysis_${BASE_FILENAME}"
    
    # Also change output file in command to log dir
    MODIFIED_COMMAND=$(echo "${command}" | sed "s|> curl\.|> ${LOG_DIR}/curl.|g")
    
    log_info "Log file: $(basename "${LOG_FILE}")"
    log_info "System memory file: $(basename "${SYSTEM_MEMORY_FILE}")"
    log_info "Analysis prefix: $(basename "${ANALYSIS_PREFIX}")"
    log_info "Modified command: ${MODIFIED_COMMAND}"
    
    # Step 1: Record baseline memory usage for 10 seconds
    log_info "Step 1: Starting baseline memory monitoring (10 seconds)..."
    pidstat -r -p "${SPIDER_PID}" 2 > "${LOG_FILE}" &
    PIDSTAT_PID=$!
    
    # Collect system-wide memory info (background)
    {
        echo "timestamp,total_memory_gb,available_memory_gb,used_memory_gb,memory_usage_percent"
        while kill -0 ${PIDSTAT_PID} 2>/dev/null; do
            TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
            MEMORY_INFO=$(free -m | awk 'NR==2{printf "%.2f,%.2f,%.2f,%.1f", $2/1024, $7/1024, $3/1024, $3*100/$2}')
            echo "${TIMESTAMP},${MEMORY_INFO}"
            sleep 2
        done
    } > "${SYSTEM_MEMORY_FILE}" &
    MEMORY_MONITOR_PID=$!
    
    # Wait 10 seconds (baseline)
    log_info "Recording baseline memory usage..."
    sleep 10
    
    # Step 2: Execute command
    log_info "Step 2: Executing command..."
    EXEC_START_TIME=$(date '+%Y-%m-%d %H:%M:%S')
    EXEC_START_TIMESTAMP=$(date +%s.%N)
    
    # Run modified command (continue on error)
    eval "${MODIFIED_COMMAND}" || log_warning "Command execution completed with warnings"
    
    EXEC_END_TIME=$(date '+%Y-%m-%d %H:%M:%S')
    EXEC_END_TIMESTAMP=$(date +%s.%N)
    
    # Calculate execution time (seconds, using awk)
    EXECUTION_TIME=$(awk "BEGIN {printf \"%.3f\", ${EXEC_END_TIMESTAMP} - ${EXEC_START_TIMESTAMP}}")
    
    log_success "Command execution completed"
    log_info "Execution time: ${EXEC_START_TIME} ~ ${EXEC_END_TIME}"
    log_info "Duration: ${EXECUTION_TIME} seconds"
    
    # Save execution time to file
    echo "${EXECUTION_TIME}" > "${LOG_DIR}/execution_time_${BASE_FILENAME}.txt"
    
    # Step 3: Record post-execution memory usage for 10 seconds
    log_info "Step 3: Recording post-execution memory usage (10 seconds)..."
    sleep 10
    
    # Stop pidstat and memory monitor processes
    kill ${PIDSTAT_PID} 2>/dev/null || true
    wait ${PIDSTAT_PID} 2>/dev/null || true
    
    kill ${MEMORY_MONITOR_PID} 2>/dev/null || true
    wait ${MEMORY_MONITOR_PID} 2>/dev/null || true
    
    log_success "Memory logging completed"
    
    # Step 4: Analyze logs
    log_info "Step 4: Analyzing memory usage..."
    
    # Wait 1 minute after analysis (minimize memory impact)
    log_info "Sleeping 60 seconds before next command to reduce memory impact..."
    sleep 60

    # Generate title per command
    CMD_TITLE=""
    if [[ "${command}" =~ vmimage ]]; then
        CMD_TITLE="VM Image List API"
    elif [[ "${command}" =~ priceinfo ]]; then
        CMD_TITLE="Price Info API"
    elif [[ "${command}" =~ spider/vm[^a-zA-Z] ]]; then
        CMD_TITLE="VM List API"
    elif [[ "${command}" =~ vpc ]]; then
        CMD_TITLE="VPC List API"
    else
        CMD_TITLE="Spider API"
    fi
    
    if [[ -n "${CONNECTION_NAME}" ]]; then
        CMD_TITLE="${CMD_TITLE} (${CONNECTION_NAME})"
    fi
    

    # Change to log dir and run analysis (pass args directly)
    (
        cd "${LOG_DIR}"
        /bin/python3 "${SCRIPT_DIR}/analyze_cb_spider_memory.py" \
            "$(basename "${LOG_FILE}")" \
            "$(basename "${SYSTEM_MEMORY_FILE}")" \
            "${SPIDER_PID}" \
            "${CMD_TITLE}" \
            "$(basename "${ANALYSIS_PREFIX}")" \
            "execution_time_$(basename "${LOG_FILE}" .log).txt" \
        || log_warning "Analysis completed with warnings"
    )
    
    log_success "Analysis completed for command ${COMMAND_COUNT}"
    log_info "Generated files:"
    log_info "  - Log: $(basename "${LOG_FILE}")"
    log_info "  - Graph: $(basename "${ANALYSIS_PREFIX}").png"
    log_info "  - Excel: $(basename "${ANALYSIS_PREFIX}").xlsx"
    
    echo ""
    
done < "${CMD_FILE}"

# Final summary
log_header "Memory Analysis Summary"
log_success "Total commands processed: ${COMMAND_COUNT}"
log_success "All log files saved to: ${LOG_DIR}"

log_info "Generated files:"
find "${LOG_DIR}" -name "*.log" -o -name "*.png" -o -name "*.xlsx" | sort | while read -r file; do
    log_info "  - $(basename "${file}")"
done

END_TIME=$(date '+%Y-%m-%d %H:%M:%S')
log_success "Analysis completed at: ${END_TIME}"
log_header "CB-Spider Server Memory Usage Analysis Finished"
