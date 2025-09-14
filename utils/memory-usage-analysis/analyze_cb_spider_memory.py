#!/usr/bin/env python3
"""
CB-Spider Server Memory Usage Pattern Analysis
Analyzes pidstat log files to generate memory usage graphs and save to Excel files
"""

import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
from datetime import datetime, timedelta
import re
import os
import seaborn as sns

# Font configuration for better compatibility
import matplotlib.font_manager as fm

def get_system_font():
    """Get available system font"""
    font_list = fm.findSystemFonts(fontpaths=None, fontext='ttf')
    preferred_fonts = []
    
    for font_path in font_list:
        try:
            font_prop = fm.FontProperties(fname=font_path)
            font_name = font_prop.get_name()
            # Check for common fonts
            if any(keyword in font_name.lower() for keyword in ['arial', 'helvetica', 'dejavu', 'liberation']):
                preferred_fonts.append(font_name)
        except:
            continue
    
    if preferred_fonts:
        return preferred_fonts[0]
    else:
        return 'sans-serif'

system_font = get_system_font()
print(f"Using font: {system_font}")

plt.rcParams['font.family'] = system_font
plt.rcParams['axes.unicode_minus'] = False

def parse_system_memory_log(filename):
    """Parse system memory log file into DataFrame"""
    try:
        df = pd.read_csv(filename)
        df['timestamp'] = pd.to_datetime(df['timestamp'])
        return df
    except Exception as e:
        print(f"❌ Failed to parse system memory log: {e}")
        return pd.DataFrame()

def parse_pidstat_log(filename, target_pid):
    """Parse pidstat log file into DataFrame"""
    data = []
    
    with open(filename, 'r') as f:
        lines = f.readlines()
    
    # Use current date as base (pidstat logs are typically from today)
    base_date = datetime.now().replace(hour=0, minute=0, second=0, microsecond=0)
    
    for line in lines[2:]:  # Skip header lines
        if line.strip() and not line.startswith('Linux') and 'UID' not in line:
            parts = line.strip().split()
            if len(parts) >= 8 and parts[2] == str(target_pid):
                try:
                    time_str = parts[0]  # HH:MM:SS format
                    time_obj = datetime.strptime(time_str, "%H:%M:%S")
                    
                    # Combine date and time
                    timestamp = base_date.replace(
                        hour=time_obj.hour,
                        minute=time_obj.minute,
                        second=time_obj.second
                    )
                    
                    # Handle crossing midnight
                    if len(data) > 0 and timestamp < data[-1]['timestamp']:
                        timestamp += timedelta(days=1)
                    
                    data.append({
                        'timestamp': timestamp,
                        'time_str': time_str,
                        'pid': int(parts[2]),
                        'minflt_per_sec': float(parts[3]),
                        'majflt_per_sec': float(parts[4]),
                        'vsz_kb': int(parts[5]),  # Virtual memory size (KB)
                        'rss_kb': int(parts[6]),  # Resident Set Size (KB)
                        'mem_percent': float(parts[7]),
                        'command': parts[8]
                    })
                except (ValueError, IndexError) as e:
                    print(f"Parsing error: {line.strip()} - {e}")
                    continue
    
    return pd.DataFrame(data)

def analyze_memory_usage(df):
    """Analyze memory usage statistics"""
    print(f"📊 CB-Spider Server (PID {df['pid'].iloc[0]}) Memory Usage Analysis")
    print("=" * 60)
    
    # Basic statistics
    print(f"📈 Analysis period: {df['timestamp'].min()} ~ {df['timestamp'].max()}")
    print(f"📊 Total data points: {len(df)}")
    print(f"⏱️  Monitoring duration: {(df['timestamp'].max() - df['timestamp'].min()).total_seconds()/60:.1f} minutes")
    print()
    
    # RSS (Physical Memory) statistics
    rss_mb = df['rss_kb'] / 1024
    print(f"🧠 RSS (Physical Memory) Usage:")
    print(f"   • Average: {rss_mb.mean():.1f} MB")
    print(f"   • Minimum: {rss_mb.min():.1f} MB")
    print(f"   • Maximum: {rss_mb.max():.1f} MB")
    print(f"   • Standard Deviation: {rss_mb.std():.1f} MB")
    print()
    
    # VSZ (Virtual Memory) statistics
    vsz_mb = df['vsz_kb'] / 1024
    print(f"💾 VSZ (Virtual Memory) Usage:")
    print(f"   • Average: {vsz_mb.mean():.1f} MB")
    print(f"   • Minimum: {vsz_mb.min():.1f} MB")
    print(f"   • Maximum: {vsz_mb.max():.1f} MB")
    print(f"   • Standard Deviation: {vsz_mb.std():.1f} MB")
    print()
    
    # Memory change pattern analysis
    rss_diff = rss_mb.diff()
    significant_increases = rss_diff[rss_diff > 5]  # Increases > 5MB
    significant_decreases = rss_diff[rss_diff < -5]  # Decreases > 5MB
    
    print(f"📈 Memory Change Patterns:")
    print(f"   • Increases >5MB: {len(significant_increases)} times")
    print(f"   • Decreases >5MB: {len(significant_decreases)} times")
    if len(significant_increases) > 0:
        print(f"   • Maximum increase: {rss_diff.max():.1f} MB")
    if len(significant_decreases) > 0:
        print(f"   • Maximum decrease: {rss_diff.min():.1f} MB")
    print()
    
    return df

def create_memory_graph(spider_df, system_df, output_file, execution_time_sec=None, command_title="CB-Spider Server Memory Usage Analysis"):
    """Generate memory usage graphs with execution time and peak annotations"""
    # Prepare time-series data
    spider_df['rss_mb'] = spider_df['rss_kb'] / 1024
    
    # Graph configuration (2 graphs)
    fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(15, 10))
    
    # Calculate execution time info for title
    time_range = (spider_df['timestamp'].max() - spider_df['timestamp'].min()).total_seconds()
    title_suffix = ""
    if execution_time_sec is not None:
        title_suffix = f" (Execution: {execution_time_sec:.1f}s)"
    
    # Combine command title with execution time
    full_title = f'{command_title}{title_suffix}'
    fig.suptitle(full_title, fontsize=16, fontweight='bold')
    
    # 1. System Total RSS Memory Usage (newly added)
    if not system_df.empty:
        ax1.plot(system_df['timestamp'], system_df['used_memory_gb'], 
                color='#2E86AB', linewidth=2, marker='o', markersize=3, label='Used Memory')

    # Show total memory size (legend only, no line)
    total_memory = system_df['total_memory_gb'].iloc[0]
    # Trick: add invisible line for legend only
    ax1.plot([], [], color='white', alpha=0.0, label=f'<Total Memory:{total_memory:.2f}GB>')

    # Find and annotate peak used memory
    peak_used_idx = system_df['used_memory_gb'].idxmax()
    peak_used_value = system_df.loc[peak_used_idx, 'used_memory_gb']
    peak_used_time = system_df.loc[peak_used_idx, 'timestamp']

    ax1.annotate(f'Peak: {peak_used_value:.1f}GB', 
                xy=(peak_used_time, peak_used_value),
                xytext=(10, 10), textcoords='offset points',
                bbox=dict(boxstyle='round,pad=0.3', facecolor='yellow', alpha=0.7),
                arrowprops=dict(arrowstyle='->', connectionstyle='arc3,rad=0'))

    # Add average line (same style as Spider)
    avg_used = system_df['used_memory_gb'].mean()
    ax1.axhline(y=avg_used, color='#2E86AB', linestyle='--', alpha=0.7, 
               label=f'System Avg: {avg_used:.2f} GB')

    ax1.set_title('System Total RSS Memory Usage', fontsize=14, fontweight='bold')
    ax1.set_ylabel('Memory Usage (GB)', fontsize=12)
    ax1.grid(True, alpha=0.3)

    # Set Y-axis scale with 50MB (0.05GB) unit margins for better visibility
    if not system_df.empty:
        min_used = system_df['used_memory_gb'].min()
        max_used = system_df['used_memory_gb'].max()

        # Use 50MB (0.05GB) unit for padding
        margin_gb = 0.05  # 50MB

        # Calculate range with 50MB margins
        y_min = max(0, min_used - margin_gb * 2)  # 100MB below minimum
        y_max = max_used + margin_gb * 2  # 100MB above maximum

        # Ensure minimum range of at least 200MB for visibility
        if y_max - y_min < 0.2:  # 200MB
            center = (y_max + y_min) / 2
            y_min = max(0, center - 0.1)  # 100MB below center
            y_max = center + 0.1  # 100MB above center

        ax1.set_ylim(bottom=y_min, top=y_max)
    else:
        ax1.set_ylim(bottom=0, top=total_memory * 1.1)

    # Custom legend: Total Memory (dummy) should be the first line
    handles, labels = ax1.get_legend_handles_labels()
    # Find the dummy handle (Total Memory) and move it to the front
    dummy_idx = None
    for i, l in enumerate(labels):
        if l.startswith('<Total Memory:'):
            dummy_idx = i
            break
    if dummy_idx is not None:
        # Move dummy handle/label to the front
        handles = [handles[dummy_idx]] + handles[:dummy_idx] + handles[dummy_idx+1:]
        labels = [labels[dummy_idx]] + labels[:dummy_idx] + labels[dummy_idx+1:]
    # Remove any duplicate labels (preserve order)
    seen = set()
    new_handles = []
    new_labels = []
    for h, l in zip(handles, labels):
        if l not in seen:
            new_handles.append(h)
            new_labels.append(l)
            seen.add(l)
    ax1.legend(new_handles, new_labels)
    
    # 2. CB-Spider RSS (Physical Memory) Usage only
    ax2.plot(spider_df['timestamp'], spider_df['rss_mb'], 
             color='#FF6B6B', linewidth=2, marker='o', markersize=3, label='CB-Spider RSS')
    ax2.set_ylabel('CB-Spider Memory Usage (MB)', fontsize=12, color='#FF6B6B')
    ax2.tick_params(axis='y', labelcolor='#FF6B6B')
    ax2.grid(True, alpha=0.3)
    
    # Set Y-axis scale with 50MB unit margins for better visibility (similar to System Memory)
    if not spider_df.empty:
        min_spider = spider_df['rss_mb'].min()
        max_spider = spider_df['rss_mb'].max()
        
        # Use 50MB unit for padding
        margin_mb = 50  # 50MB
        
        # Calculate range with 50MB margins
        y_min = max(0, min_spider - margin_mb)  # 50MB below minimum
        y_max = max_spider + margin_mb  # 50MB above maximum
        
        # Ensure minimum range of at least 100MB for visibility
        if y_max - y_min < 100:  # 100MB
            center = (y_max + y_min) / 2
            y_min = max(0, center - 50)  # 50MB below center
            y_max = center + 50  # 50MB above center
        
        ax2.set_ylim(bottom=y_min, top=y_max)
    else:
        ax2.set_ylim(bottom=0)
    
    # Add average line
    avg_rss = spider_df['rss_mb'].mean()
    ax2.axhline(y=avg_rss, color='#FF6B6B', linestyle='--', alpha=0.7, 
               label=f'Spider Avg: {avg_rss:.1f} MB')
    
    # Find and annotate peak Spider memory
    peak_spider_idx = spider_df['rss_mb'].idxmax()
    peak_spider_value = spider_df.loc[peak_spider_idx, 'rss_mb']
    peak_spider_time = spider_df.loc[peak_spider_idx, 'timestamp']
    
    ax2.annotate(f'Peak: {peak_spider_value:.1f}MB', 
                xy=(peak_spider_time, peak_spider_value),
                xytext=(10, 10), textcoords='offset points',
                bbox=dict(boxstyle='round,pad=0.3', facecolor='orange', alpha=0.7),
                arrowprops=dict(arrowstyle='->', connectionstyle='arc3,rad=0'))
    
    ax2.set_title('CB-Spider Memory Usage', fontsize=14, fontweight='bold')
    ax2.set_xlabel('Time', fontsize=12)
    ax2.legend()
    
    # Time axis format configuration
    for ax in [ax1, ax2]:
        # Calculate time range
        time_range = (spider_df['timestamp'].max() - spider_df['timestamp'].min()).total_seconds()
        print(f"🕐 Time range: {time_range:.1f} seconds ({time_range/60:.1f} minutes)")
        
        # Set appropriate intervals based on time range
        if time_range <= 60:  # <= 1 minute
            ax.xaxis.set_major_formatter(mdates.DateFormatter('%H:%M:%S'))
            ax.xaxis.set_major_locator(mdates.SecondLocator(interval=10))
        elif time_range <= 300:  # <= 5 minutes
            ax.xaxis.set_major_formatter(mdates.DateFormatter('%H:%M:%S'))
            ax.xaxis.set_major_locator(mdates.SecondLocator(interval=30))
        elif time_range <= 1800:  # <= 30 minutes
            ax.xaxis.set_major_formatter(mdates.DateFormatter('%H:%M'))
            ax.xaxis.set_major_locator(mdates.MinuteLocator(interval=5))
        elif time_range <= 3600:  # <= 1 hour
            ax.xaxis.set_major_formatter(mdates.DateFormatter('%H:%M'))
            ax.xaxis.set_major_locator(mdates.MinuteLocator(interval=10))
        elif time_range <= 7200:  # <= 2 hours
            ax.xaxis.set_major_formatter(mdates.DateFormatter('%H:%M'))
            ax.xaxis.set_major_locator(mdates.MinuteLocator(interval=20))
        else:  # > 2 hours
            ax.xaxis.set_major_formatter(mdates.DateFormatter('%H:%M'))
            ax.xaxis.set_major_locator(mdates.MinuteLocator(interval=30))
        
        # Limit tick count (max 15)
        ax.locator_params(axis='x', nbins=15)
        plt.setp(ax.xaxis.get_majorticklabels(), rotation=45)
    
    plt.tight_layout()
    plt.savefig(output_file, dpi=300, bbox_inches='tight')
    print(f"📊 Graph saved: {output_file}")
    return fig

def save_to_excel(spider_df, system_df, output_file):
    """Save data to Excel file"""
    # Prepare time-series data
    export_df = spider_df.copy()
    export_df['rss_mb'] = export_df['rss_kb'] / 1024
    export_df['vsz_mb'] = export_df['vsz_kb'] / 1024
    export_df['timestamp_str'] = export_df['timestamp'].dt.strftime('%Y-%m-%d %H:%M:%S')
    
    # Column organization
    columns_to_export = [
        'timestamp_str', 'time_str', 'pid', 'command',
        'rss_kb', 'rss_mb', 'vsz_kb', 'vsz_mb', 'mem_percent',
        'minflt_per_sec', 'majflt_per_sec'
    ]
    
    export_df = export_df[columns_to_export]
    
    # English column names
    export_df.columns = [
        'Timestamp', 'Time', 'PID', 'Command',
        'RSS(KB)', 'RSS(MB)', 'VSZ(KB)', 'VSZ(MB)', 'Memory%',
        'MinorFaults/sec', 'MajorFaults/sec'
    ]
    
    # Save to Excel file
    with pd.ExcelWriter(output_file, engine='openpyxl') as writer:
        # CB-Spider memory data
        export_df.to_excel(writer, sheet_name='CB-Spider Memory', index=False)
        
        # System memory data
        if not system_df.empty:
            system_export_df = system_df.copy()
            system_export_df['timestamp_str'] = system_export_df['timestamp'].dt.strftime('%Y-%m-%d %H:%M:%S')
            system_columns = ['timestamp_str', 'total_memory_gb', 'available_memory_gb', 'used_memory_gb', 'memory_usage_percent']
            system_export_df = system_export_df[system_columns]
            system_export_df.columns = ['Timestamp', 'Total Memory(GB)', 'Available Memory(GB)', 'Used Memory(GB)', 'System Memory Usage(%)']
            system_export_df.to_excel(writer, sheet_name='System Memory', index=False)
        
        # Statistical summary
        stats_data = {
            'Metric': ['Spider RSS Average(MB)', 'Spider RSS Minimum(MB)', 'Spider RSS Maximum(MB)', 'Spider RSS StdDev(MB)',
                      'Spider Memory% Average', 'Spider Memory% Minimum', 'Spider Memory% Maximum',
                      'Total Data Points', 'Monitoring Duration(min)'],
            'Value': [
                round(export_df['RSS(MB)'].mean(), 2),
                round(export_df['RSS(MB)'].min(), 2),
                round(export_df['RSS(MB)'].max(), 2),
                round(export_df['RSS(MB)'].std(), 2),
                round(export_df['Memory%'].mean(), 3),
                round(export_df['Memory%'].min(), 3),
                round(export_df['Memory%'].max(), 3),
                len(export_df),
                round((spider_df['timestamp'].max() - spider_df['timestamp'].min()).total_seconds()/60, 1)
            ]
        }
        
        if not system_df.empty:
            stats_data['Metric'].extend(['System Total Memory(GB)', 'System Average Usage(%)', 'System Maximum Usage(%)'])
            stats_data['Value'].extend([
                round(system_df['total_memory_gb'].iloc[0], 1),
                round(system_df['memory_usage_percent'].mean(), 1),
                round(system_df['memory_usage_percent'].max(), 1)
            ])
        
        stats_df = pd.DataFrame(stats_data)
        stats_df.to_excel(writer, sheet_name='Statistics', index=False)
    
    print(f"📋 Excel file saved: {output_file}")

def main(log_file=None, system_memory_file=None, target_pid=None, command_title=None, output_prefix=None, execution_time_file=None):
    """Main function with command line arguments support"""
    # Default values
    if log_file is None:
        log_file = 'pidstat.2sec.log'
    if system_memory_file is None:
        system_memory_file = 'system_memory.log'
    if target_pid is None:
        target_pid = 1292676
    if output_prefix is None:
        output_prefix = 'cb_spider_memory_analysis_pid_1292676'
    if command_title is None:
        command_title = "CB-Spider Server Memory Usage Analysis"
    
    # Load execution time
    execution_time_sec = None
    if execution_time_file and os.path.exists(execution_time_file):
        with open(execution_time_file, "r") as f:
            execution_time_sec = float(f.read().strip())
    else:
        # Fallback: find matching execution time file
        base_name = os.path.basename(log_file).replace('.txt', '').replace('pidstat_', '')
        fallback_execution_time_file = f'execution_time_{base_name}.txt'
        
        if os.path.exists(fallback_execution_time_file):
            with open(fallback_execution_time_file, "r") as f:
                execution_time_sec = float(f.read().strip())
        else:
            # Try to find any matching execution time file
            execution_time_files = [f for f in os.listdir('.') if f.startswith('execution_time_') and f.endswith('.txt')]
            if execution_time_files:
                with open(execution_time_files[0], "r") as f:
                    execution_time_sec = float(f.read().strip())
    
    if not os.path.exists(log_file):
        print(f"❌ Log file not found: {log_file}")
        return
    
    print("🔍 Starting pidstat log file analysis...")
    
    # Parse pidstat log
    spider_df = parse_pidstat_log(log_file, target_pid)
    
    if spider_df.empty:
        print(f"❌ No data found for PID {target_pid}.")
        return
    
    # Data validation
    time_range = (spider_df['timestamp'].max() - spider_df['timestamp'].min()).total_seconds()
    print(f"📊 Spider data: {len(spider_df)} entries, time range: {time_range:.1f}s ({time_range/60:.1f}min)")
    
    if time_range > 86400:  # Warn if over 1 day (not an error)
        print(f"⚠️ Very large time range: {time_range:.1f}s ({time_range/3600:.1f}h)")
        print("⚠️ Graph generation may take a long time.")
    
    # Parse system memory log
    system_df = pd.DataFrame()
    if os.path.exists(system_memory_file):
        system_df = parse_system_memory_log(system_memory_file)
        print(f"📊 System memory log parsing completed: {len(system_df)} entries")
    else:
        print(f"⚠️ System memory log file not found: {system_memory_file}")
    
    # Analyze memory usage
    spider_df = analyze_memory_usage(spider_df)
    
    # Create graph
    graph_file = f'{output_prefix}.png'
    create_memory_graph(spider_df, system_df, graph_file, execution_time_sec, command_title)
    
    # Save Excel file
    excel_file = f'{output_prefix}.xlsx'
    save_to_excel(spider_df, system_df, excel_file)
    
    print("\n✅ Analysis completed!")
    print(f"📊 Graph: {graph_file}")
    print(f"📋 Excel: {excel_file}")

if __name__ == "__main__":
    import sys
    
    # Parse command line arguments
    if len(sys.argv) >= 7:
        # Full argument mode: log_file system_memory_file target_pid command_title output_prefix execution_time_file
        log_file = sys.argv[1]
        system_memory_file = sys.argv[2]
        target_pid = int(sys.argv[3])
        command_title = sys.argv[4]
        output_prefix = sys.argv[5]
        execution_time_file = sys.argv[6]
        main(log_file, system_memory_file, target_pid, command_title, output_prefix, execution_time_file)
    elif len(sys.argv) >= 2:
        # Legacy mode: just command title
        command_title = sys.argv[1]
        main(command_title=command_title)
    else:
        # Default mode
        main()
