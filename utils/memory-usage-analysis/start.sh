#!/bin/bash

# CB-Spider 서버 메모리 사용량 자동 분석 스크립트
# 작성자: Auto Memory Analysis Script
# 작성일: $(date '+%Y-%m-%d %H:%M:%S')

set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 로그 함수
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

# 스크립트 시작
log_header "CB-Spider Server Memory Usage Analysis Started"
START_TIME=$(date '+%Y%m%d_%H%M%S')

# 현재 디렉토리 확인

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}" )" && pwd)"
LOG_DIR="${SCRIPT_DIR}/log-${START_TIME}"

log_info "Script Directory: ${SCRIPT_DIR}"
log_info "Log Directory: ${LOG_DIR}"

# 로그 디렉토리 생성 (버전 파일보다 먼저)
mkdir -p "${LOG_DIR}"
# CB-Spider 버전 정보 저장
SPIDER_VERSION_FILE="${LOG_DIR}/spider_version.txt"
if [ -x "${SCRIPT_DIR}/../../bin/cb-spider" ]; then
    "${SCRIPT_DIR}/../../bin/cb-spider" -v > "${SPIDER_VERSION_FILE}" 2>&1 || true
elif [ -x "${SCRIPT_DIR}/../../bin/spider" ]; then
    "${SCRIPT_DIR}/../../bin/spider" -v > "${SPIDER_VERSION_FILE}" 2>&1 || true
fi
log_info "Spider version info saved to: ${SPIDER_VERSION_FILE}"
# CB-Spider PID 읽기 (스크립트 기준 상위 2단계 bin 디렉토리)
PID_FILE="${SCRIPT_DIR}/../../bin/spider.pid"
if [ ! -f "${PID_FILE}" ]; then
    log_error "PID file not found: ${PID_FILE}"
    log_error "Please make sure CB-Spider server is running"
    exit 1
fi

SPIDER_PID=$(cat "${PID_FILE}")
log_info "CB-Spider PID: ${SPIDER_PID}"

# PID가 실제로 실행 중인지 확인
if ! ps -p "${SPIDER_PID}" > /dev/null 2>&1; then
    log_error "CB-Spider process (PID: ${SPIDER_PID}) is not running"
    exit 1
fi

log_success "CB-Spider server is running (PID: ${SPIDER_PID})"

# 명령어 파일 확인 - 명령행 인자가 있으면 사용, 없으면 환경변수나 기본값 사용
if [ $# -gt 0 ]; then
    # 첫 번째 인자를 명령어 파일로 사용
    if [[ "$1" == /* ]]; then
        # 절대 경로인 경우 그대로 사용
        CMD_FILE="$1"
    else
        # 상대 경로인 경우 스크립트 디렉토리 기준으로 변환
        CMD_FILE="${SCRIPT_DIR}/$1"
    fi
else
    # 인자가 없으면 환경변수나 기본값 사용
    CMD_FILE="${CMD_FILE:-${SCRIPT_DIR}/cmd-list.cmd}"
fi

if [ ! -f "${CMD_FILE}" ]; then
    log_error "Command file not found: ${CMD_FILE}"
    exit 1
fi

log_info "Command file: ${CMD_FILE}"

# 명령어 개수 확인 (# 또는 // 주석과 빈 줄 제외)
TOTAL_COMMANDS=$(grep -v -E "^[[:space:]]*#|^[[:space:]]*//|^[[:space:]]*$" "${CMD_FILE}" | wc -l)
log_info "Total commands to execute: ${TOTAL_COMMANDS}"

# 로그 디렉토리 생성
mkdir -p "${LOG_DIR}"

# 명령어 실행 및 분석
COMMAND_COUNT=0
while IFS= read -r command || [ -n "$command" ]; do
    # 빈 줄, 공백만 있는 줄, 주석 줄 건너뛰기 (# 또는 // 로 시작하는 라인)
    if [[ -z "${command}" || "${command}" =~ ^[[:space:]]*$ || "${command}" =~ ^[[:space:]]*# || "${command}" =~ ^[[:space:]]*// ]]; then
        continue
    fi
    
    COMMAND_COUNT=$((COMMAND_COUNT + 1))
    CMD_START_TIME=$(date '+%Y%m%d_%H%M%S')
    
    log_header "Command ${COMMAND_COUNT}/${TOTAL_COMMANDS}: Memory Analysis"
    log_info "Command: ${command}"
    
    # ConnectionName 추출 (URL 파라미터 또는 JSON 데이터에서)
    CONNECTION_NAME=""
    if [[ "${command}" =~ ConnectionName=([^[:space:]&]+) ]]; then
        # URL 파라미터에서 추출
        CONNECTION_NAME="${BASH_REMATCH[1]}"
    elif [[ "${command}" =~ \"ConnectionName\"[[:space:]]*:[[:space:]]*\"([^\"]+)\" ]]; then
        # JSON 데이터에서 추출
        CONNECTION_NAME="${BASH_REMATCH[1]}"
    fi
    
    # 파일명 생성 (특수문자 제거하고 ConnectionName 포함)
    CMD_SAFE=$(echo "${command}" | sed 's/[^a-zA-Z0-9._-]/_/g' | cut -c1-50)
    if [[ -n "${CONNECTION_NAME}" ]]; then
        CMD_SAFE="${CMD_SAFE}_ConnectionName_${CONNECTION_NAME}"
    fi
    BASE_FILENAME="${CMD_SAFE}_${CMD_START_TIME}"
    
    LOG_FILE="${LOG_DIR}/${BASE_FILENAME}.log"
    SYSTEM_MEMORY_FILE="${LOG_DIR}/system_memory_${BASE_FILENAME}.log"
    ANALYSIS_PREFIX="${LOG_DIR}/analysis_${BASE_FILENAME}"
    
    # 명령어 출력 파일도 로그 디렉토리로 수정
    MODIFIED_COMMAND=$(echo "${command}" | sed "s|> curl\.|> ${LOG_DIR}/curl.|g")
    
    log_info "Log file: $(basename "${LOG_FILE}")"
    log_info "System memory file: $(basename "${SYSTEM_MEMORY_FILE}")"
    log_info "Analysis prefix: $(basename "${ANALYSIS_PREFIX}")"
    log_info "Modified command: ${MODIFIED_COMMAND}"
    
    # Step 1: 평상시 메모리 사용량 10초간 기록 시작
    log_info "Step 1: Starting baseline memory monitoring (10 seconds)..."
    pidstat -r -p "${SPIDER_PID}" 2 > "${LOG_FILE}" &
    PIDSTAT_PID=$!
    
    # 시스템 전체 메모리 정보 수집 (백그라운드)
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
    
    # 10초 대기 (평상시 상태)
    log_info "Recording baseline memory usage..."
    sleep 10
    
    # Step 2: 명령 실행
    log_info "Step 2: Executing command..."
    EXEC_START_TIME=$(date '+%Y-%m-%d %H:%M:%S')
    EXEC_START_TIMESTAMP=$(date +%s.%N)
    
    # 수정된 명령 실행 (에러가 있어도 계속 진행)
    eval "${MODIFIED_COMMAND}" || log_warning "Command execution completed with warnings"
    
    EXEC_END_TIME=$(date '+%Y-%m-%d %H:%M:%S')
    EXEC_END_TIMESTAMP=$(date +%s.%N)
    
    # 실행 시간 계산 (초 단위) - awk 사용
    EXECUTION_TIME=$(awk "BEGIN {printf \"%.3f\", ${EXEC_END_TIMESTAMP} - ${EXEC_START_TIMESTAMP}}")
    
    log_success "Command execution completed"
    log_info "Execution time: ${EXEC_START_TIME} ~ ${EXEC_END_TIME}"
    log_info "Duration: ${EXECUTION_TIME} seconds"
    
    # 실행 시간 정보를 파일에 저장
    echo "${EXECUTION_TIME}" > "${LOG_DIR}/execution_time_${BASE_FILENAME}.txt"
    
    # Step 3: 명령 완료 후 10초간 더 기록
    log_info "Step 3: Recording post-execution memory usage (10 seconds)..."
    sleep 10
    
    # pidstat과 메모리 모니터링 프로세스 종료
    kill ${PIDSTAT_PID} 2>/dev/null || true
    wait ${PIDSTAT_PID} 2>/dev/null || true
    
    kill ${MEMORY_MONITOR_PID} 2>/dev/null || true
    wait ${MEMORY_MONITOR_PID} 2>/dev/null || true
    
    log_success "Memory logging completed"
    
    # Step 4: 로그 분석
    log_info "Step 4: Analyzing memory usage..."
    
        # 명령 분석 후 1분 대기 (메모리 영향 최소화)
        log_info "Sleeping 60 seconds before next command to reduce memory impact..."
        sleep 60

    # 명령어별 타이틀 생성
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
    

    # 현재 디렉토리를 로그 디렉토리로 변경하여 분석 실행 (인자 직접 전달)
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

# 최종 결과 요약
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
