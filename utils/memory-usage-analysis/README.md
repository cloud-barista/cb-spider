# CB-Spider Memory Usage Analysis Utility

This utility provides automated measurement, logging, and reporting of CB-Spider server memory usage for various API calls. It helps diagnose memory consumption patterns and performance bottlenecks by generating visual graphs and Excel reports.

## Features
- Automated execution of API command lists and memory monitoring
- Collection of both CB-Spider process and system memory usage
- Analysis scripts to generate summary statistics, graphs (PNG), and Excel reports (XLSX)
- HTML report generation for easy result browsing

## Usage

### 1. Prerequisites
- Python 3.x with `pandas`, `matplotlib`, `seaborn`, and `openpyxl` installed
- Linux environment with `pidstat`, `free`, and standard shell utilities

### 2. How to Run

1. **Start the analysis:**
   ```sh
   ./start.sh [command-list-file]
   ```
   - If no argument is given, `cmd-list.cmd` in the script directory is used by default.
   - The script will create a timestamped log directory and run all commands, collecting memory usage data.

2. **Generate the HTML report:**
   ```sh
   ./report.sh <log-dir>
   ```
   - Replace `<log-dir>` with the log directory created by `start.sh` (e.g., `log-20250912_153000`).
   - The report will be saved as `index.html` in the log directory.

### 3. Output
- Per-command logs, memory usage graphs (`.png`), Excel summaries (`.xlsx`), and a consolidated HTML report.

## File Overview
- `start.sh`: Main automation script for running commands and collecting memory data
- `analyze_cb_spider_memory.py`: Python script for analyzing and visualizing memory usage
- `report.sh`: Generates an HTML summary report from collected data

## Example
```sh
./start.sh
./report.sh log-20250912_153000
```

## Notes
- Ensure CB-Spider server is running before starting analysis.
- Review and customize `cmd-list.cmd` to match your API test scenarios.
