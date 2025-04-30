package commonruntime

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"syscall"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

//================ CB-Spider System Usage Info Handler

// SystemInfo contains all system information
type SystemInfo struct {
	Hostname        string
	Platform        string
	PlatformVersion string
	KernelArch      string
	KernelVersion   string
	Uptime          string

	CPUModel      string
	PhysicalCores int
	LogicalCores  int
	ClockSpeed    string

	TotalMemory string
	SwapMemory  string

	DiskPartitions []DiskPartitionInfo
}

// DiskPartitionInfo contains information about a disk partition
type DiskPartitionInfo struct {
	MountPoint string
	TotalSpace string
}

// ResourceUsage contains system resource usage information
type ResourceUsage struct {
	// System information
	SystemCPUPercent     string            // Total CPU usage with % format
	SystemCPUCorePercent map[string]string // Per-core CPU usage percentages with core number as key
	SystemMemoryUsed     string            // GiB format
	SystemMemoryTotal    string            // GiB format
	SystemMemoryPercent  string            // % format
	SystemDiskRead       string
	SystemDiskWrite      string
	SystemNetSent        string
	SystemNetReceived    string

	// Process information
	ProcessName           string
	ProcessCPUPercent     string            // Total process CPU usage with % format
	ProcessCPUCorePercent map[string]string // Per-core process CPU usage percentages with core number as key
	ProcessMemoryUsed     string            // GiB format
	ProcessMemoryPercent  string            // % format
	ProcessDiskRead       string
	ProcessDiskWrite      string
	ProcessNetSent        string
	ProcessNetReceived    string
}

const (
	// Error message templates
	ERR_GETTING_PROCESS_INFO    = "error getting process information: %v"
	ERR_GETTING_HOST_INFO       = "error getting host information: %v"
	ERR_GETTING_CPU_INFO        = "error getting CPU information: %v"
	ERR_GETTING_MEMORY_INFO     = "error getting memory information: %v"
	ERR_GETTING_DISK_INFO       = "error getting disk information: %v"
	ERR_GETTING_DISK_IO_INFO    = "error getting disk I/O information: %v"
	ERR_GETTING_NETWORK_IO_INFO = "error getting network I/O information: %v"
)

// FetchSystemInfo collects and returns system information
func FetchSystemInfo() (SystemInfo, error) {
	cblog.Info("Collecting system information")

	sysInfo, err := getSystemInfo()
	if err != nil {
		cblog.Error(err)
		return SystemInfo{}, err
	}

	cblog.Info("System information collected successfully")
	return sysInfo, nil
}

// FetchResourceUsage collects and returns resource usage information
func FetchResourceUsage() (ResourceUsage, error) {
	cblog.Info("Collecting resource usage information")

	// Get current process ID
	pid := os.Getpid()
	currentProcess, err := process.NewProcess(int32(pid))
	if err != nil {
		cblog.Error(fmt.Errorf(ERR_GETTING_PROCESS_INFO, err))
		return ResourceUsage{}, fmt.Errorf(ERR_GETTING_PROCESS_INFO, err)
	}

	resourceUsage, err := getResourceUsage(currentProcess)
	if err != nil {
		cblog.Error(err)
		return ResourceUsage{}, err
	}

	cblog.Info("Resource usage information collected successfully")
	return resourceUsage, nil
}

// DisplaySystemInfo prints the collected system information
func DisplaySystemInfo(sysInfo *SystemInfo) {
	if nil == sysInfo {
		cblog.Error(fmt.Errorf("cannot display empty SystemInfo"))
		return
	}

	printSystemInfoTable(sysInfo)
}

// DisplayResourceUsage prints the collected resource usage information
func DisplayResourceUsage(sysInfoTotalMemory string, usage *ResourceUsage) {
	if nil == usage {
		cblog.Error(fmt.Errorf("cannot display empty ResourceUsage"))
		return
	}

	printResourceUsageTable(sysInfoTotalMemory, usage)
}

// Convert bytes to GiB
func bytesToGiB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024 * 1024)
}

// Convert MHz to GHz
func mhzToGHz(mhz float64) float64 {
	return mhz / 1000.0
}

// getSystemInfo collects system information
func getSystemInfo() (SystemInfo, error) {
	cblog.Info("Collecting system information")

	var sysInfo SystemInfo

	// Get host information
	hostInfo, err := host.Info()
	if err != nil {
		return SystemInfo{}, fmt.Errorf(ERR_GETTING_HOST_INFO, err)
	}

	sysInfo.Hostname = hostInfo.Hostname
	sysInfo.Platform = hostInfo.Platform
	sysInfo.PlatformVersion = hostInfo.PlatformVersion
	sysInfo.KernelArch = hostInfo.KernelArch
	sysInfo.KernelVersion = hostInfo.KernelVersion
	sysInfo.Uptime = formatUptime(hostInfo.Uptime)

	// Get CPU information
	cpuInfo, err := cpu.Info()
	if err != nil || len(cpuInfo) == 0 {
		cblog.Warn("cpu.Info() failed or returned no info. Trying fallback...")
		sysInfo.CPUModel = "Unknown"
		sysInfo.ClockSpeed = "Unknown"
	} else {
		sysInfo.CPUModel = cpuInfo[0].ModelName
		sysInfo.ClockSpeed = fmt.Sprintf("%.2f GHz", mhzToGHz(cpuInfo[0].Mhz)) // Convert MHz to GHz
	}

	if len(cpuInfo) > 0 {
		sysInfo.CPUModel = cpuInfo[0].ModelName
		sysInfo.ClockSpeed = fmt.Sprintf("%.2f GHz", mhzToGHz(cpuInfo[0].Mhz)) // Convert MHz to GHz
	}

	physicalCores, err := cpu.Counts(false)
	if err != nil {
		cblog.Error(fmt.Errorf("error getting physical CPU cores: %v", err))
		// Continue despite error
	}

	logicalCores, err := cpu.Counts(true)
	if err != nil {
		cblog.Error(fmt.Errorf("error getting logical CPU cores: %v", err))
		// Continue despite error
	}

	sysInfo.PhysicalCores = physicalCores
	sysInfo.LogicalCores = logicalCores

	// Get memory information
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return SystemInfo{}, fmt.Errorf(ERR_GETTING_MEMORY_INFO, err)
	}

	sysInfo.TotalMemory = fmt.Sprintf("%.2f GiB", bytesToGiB(memInfo.Total))    // Convert to GiB
	sysInfo.SwapMemory = fmt.Sprintf("%.2f GiB", bytesToGiB(memInfo.SwapTotal)) // Convert to GiB

	// Get disk information
	diskParts, err := disk.Partitions(false)
	if err != nil {
		return SystemInfo{}, fmt.Errorf(ERR_GETTING_DISK_INFO, err)
	}

	for _, part := range diskParts {
		usage, err := disk.Usage(part.Mountpoint)
		if err != nil {
			cblog.Error(fmt.Errorf("error getting usage for disk partition %s: %v", part.Mountpoint, err))
			continue
		}

		partInfo := DiskPartitionInfo{
			MountPoint: part.Mountpoint,
			TotalSpace: formatBytes(usage.Total),
		}
		sysInfo.DiskPartitions = append(sysInfo.DiskPartitions, partInfo)
	}

	return sysInfo, nil
}

// getResourceUsage collects resource usage information
func getResourceUsage(currentProcess *process.Process) (ResourceUsage, error) {
	cblog.Info("Collecting resource usage information")

	var usage ResourceUsage

	// Initialize maps for CPU core usage
	usage.SystemCPUCorePercent = make(map[string]string)
	usage.ProcessCPUCorePercent = make(map[string]string)

	// Get logical core count for CPU usage normalization
	logicalCores, err := cpu.Counts(true)
	if err != nil {
		logicalCores = 1 // Default to 1 to avoid division by zero
		cblog.Error(fmt.Errorf("error getting logical CPU cores: %v", err))
	}

	// Get process name
	processName, err := currentProcess.Name()
	if err != nil {
		cblog.Error(fmt.Errorf("error getting process name: %v", err))
		usage.ProcessName = "Unknown"
	} else {
		usage.ProcessName = processName
	}

	// Get first measurements - disk and network I/O
	diskIOCountersBefore, err := disk.IOCounters()
	if err != nil {
		return ResourceUsage{}, fmt.Errorf(ERR_GETTING_DISK_IO_INFO, err)
	}

	// Get process I/O before
	var processReadBytesBefore, processWriteBytesBefore uint64 = 0, 0
	if runtime.GOOS != "darwin" {
		processIOStatBefore, err := currentProcess.IOCounters()
		if err != nil {
			cblog.Error(fmt.Errorf("error getting process I/O counters: %v", err))
		} else {
			processReadBytesBefore = processIOStatBefore.ReadBytes
			processWriteBytesBefore = processIOStatBefore.WriteBytes
		}
	} else {
		cblog.Warn("Process.IOCounters() is not implemented on macOS, skipping.")
	}

	// Get first network I/O measurement
	netIOCountersBefore, err := net.IOCounters(false) // false means don't separate by interface
	if err != nil {
		return ResourceUsage{}, fmt.Errorf(ERR_GETTING_NETWORK_IO_INFO, err)
	}

	// Wait for a measurement interval - use the same interval for all measurements
	time.Sleep(time.Second)

	// Now get CPU usage - Overall
	cpuPercent, err := cpu.Percent(0, false) // Use 0 since we already waited above
	if err != nil {
		return ResourceUsage{}, fmt.Errorf(ERR_GETTING_CPU_INFO, err)
	}

	if len(cpuPercent) > 0 {
		usage.SystemCPUPercent = fmt.Sprintf("%.2f %%", cpuPercent[0]) // Format with %
	}

	// Get CPU usage - Per Core
	perCoreCPUPercent, err := cpu.Percent(0, true) // Use 0 since we already waited above
	if err != nil {
		cblog.Error(fmt.Errorf("error getting per-core CPU usage: %v", err))
	} else {
		for i, corePercent := range perCoreCPUPercent {
			// Format with padding zeros for proper sorting in JSON
			// Core0 -> Core00, Core1 -> Core01, ..., Core10 -> Core10
			coreKey := fmt.Sprintf("Core%02d", i)
			usage.SystemCPUCorePercent[coreKey] = fmt.Sprintf("%.2f %%", corePercent)
		}
	}

	// Get process CPU usage - Overall
	processCPU, err := currentProcess.CPUPercent()
	if err != nil {
		cblog.Error(fmt.Errorf("error getting process CPU usage: %v", err))
		// Continue despite error
	} else {
		// Normalize process CPU usage by dividing by logical core count
		// This makes it comparable to system CPU percent
		if logicalCores > 0 {
			normalizedProcessCPU := processCPU / float64(logicalCores)
			usage.ProcessCPUPercent = fmt.Sprintf("%.2f %%", normalizedProcessCPU)
		} else {
			usage.ProcessCPUPercent = fmt.Sprintf("%.2f %%", processCPU)
		}
	}

	// Get process CPU usage - Per Core
	if len(perCoreCPUPercent) > 0 && processCPU > 0 {
		// Estimate based on core activity
		// This approach avoids the inflated percentages
		for i, corePercent := range perCoreCPUPercent {
			// Calculate what portion of this core the process might be using
			// Assume process is using cores proportional to their system load
			coreProcessPercent := 0.0
			if corePercent > 0 && logicalCores > 0 {
				// Cap the per-core usage at the core's system usage
				coreRatio := corePercent / 100.0 // Convert percent to ratio
				if coreRatio > 0 {
					// Estimate core usage, but don't exceed the core's available capacity
					estProcessOnCore := (processCPU / 100.0) * (coreRatio * float64(logicalCores))
					coreProcessPercent = math.Min(corePercent, estProcessOnCore)
				}
			}
			// Format with padding zeros for proper sorting in JSON
			coreKey := fmt.Sprintf("Core%02d", i)
			usage.ProcessCPUCorePercent[coreKey] = fmt.Sprintf("%.2f %%", coreProcessPercent)
		}
	}

	// Get memory usage
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return ResourceUsage{}, fmt.Errorf(ERR_GETTING_MEMORY_INFO, err)
	}

	usage.SystemMemoryUsed = fmt.Sprintf("%.2f GiB", bytesToGiB(memInfo.Used))   // Convert to GiB
	usage.SystemMemoryTotal = fmt.Sprintf("%.2f GiB", bytesToGiB(memInfo.Total)) // Convert to GiB
	usage.SystemMemoryPercent = fmt.Sprintf("%.2f %%", memInfo.UsedPercent)      // Format with %

	processMemory, err := currentProcess.MemoryInfo()
	if err != nil {
		cblog.Error(fmt.Errorf("error getting process memory usage: %v", err))
		// Continue despite error
	} else if processMemory != nil {
		usage.ProcessMemoryUsed = fmt.Sprintf("%.2f GiB", bytesToGiB(processMemory.RSS)) // Convert to GiB
		// Calculate percentage
		if memInfo.Total > 0 {
			procMemPercent := (float64(processMemory.RSS) / float64(memInfo.Total)) * 100
			usage.ProcessMemoryPercent = fmt.Sprintf("%.2f %%", procMemPercent) // Format with %
		}
	}

	// Get process I/O after the interval
	if runtime.GOOS != "darwin" {
		processIOStatAfter, err := currentProcess.IOCounters()
		if err != nil {
			cblog.Error(fmt.Errorf("error getting process I/O counters after interval: %v", err))
		} else {
			processReadRate := processIOStatAfter.ReadBytes - processReadBytesBefore
			processWriteRate := processIOStatAfter.WriteBytes - processWriteBytesBefore
			usage.ProcessDiskRead = formatBytes(processReadRate) + "/s"
			usage.ProcessDiskWrite = formatBytes(processWriteRate) + "/s"
		}
	} else {
		usage.ProcessDiskRead = "NA"
		usage.ProcessDiskWrite = "NA"
	}

	// Get second disk I/O measurement
	diskIOCountersAfter, err := disk.IOCounters()
	if err != nil {
		return ResourceUsage{}, fmt.Errorf(ERR_GETTING_DISK_IO_INFO, err)
	}

	var totalSystemDiskRead, totalSystemDiskWrite uint64
	for diskName, diskIOAfter := range diskIOCountersAfter {
		if diskIOBefore, exists := diskIOCountersBefore[diskName]; exists {
			totalSystemDiskRead += diskIOAfter.ReadBytes - diskIOBefore.ReadBytes
			totalSystemDiskWrite += diskIOAfter.WriteBytes - diskIOBefore.WriteBytes
		}
	}
	usage.SystemDiskRead = formatBytes(totalSystemDiskRead) + "/s"
	usage.SystemDiskWrite = formatBytes(totalSystemDiskWrite) + "/s"

	// Get second network I/O measurement
	netIOCountersAfter, err := net.IOCounters(false)
	if err != nil {
		return ResourceUsage{}, fmt.Errorf(ERR_GETTING_NETWORK_IO_INFO, err)
	}

	if len(netIOCountersBefore) > 0 && len(netIOCountersAfter) > 0 {
		// Using index 0 because we're getting the all-interfaces stats (not per-interface)
		systemNetSent := netIOCountersAfter[0].BytesSent - netIOCountersBefore[0].BytesSent
		systemNetReceived := netIOCountersAfter[0].BytesRecv - netIOCountersBefore[0].BytesRecv

		usage.SystemNetSent = formatBytes(systemNetSent) + "/s"
		usage.SystemNetReceived = formatBytes(systemNetReceived) + "/s"

		// Get process connections to estimate network I/O
		connections, err := currentProcess.Connections()
		if err != nil {
			cblog.Error(fmt.Errorf("error getting process connections: %v", err))
			// If we can't get process connections, make a conservative estimate
			// based on active TCP/UDP connections
			usage.ProcessNetSent = formatBytes(systemNetSent/10) + "/s"
			usage.ProcessNetReceived = formatBytes(systemNetReceived/10) + "/s"
		} else {
			// Improve the estimation of process network usage
			// Count active connections (those that are likely transferring data)
			activeConnections := 0
			for _, conn := range connections {
				// Count established TCP connections and UDP connections
				if (conn.Type == syscall.SOCK_STREAM && conn.Status == "ESTABLISHED") ||
					conn.Type == syscall.SOCK_DGRAM {
					activeConnections++
				}
			}

			// Calculate a more realistic connection ratio
			connectionRatio := 0.0
			if activeConnections > 0 {
				// Use a more meaningful formula - estimate based on active connections
				// and considering other processes on the system may have connections too
				totalSystemEstimatedConns := float64(activeConnections) * 2.0 // estimate total system connections
				if totalSystemEstimatedConns > 0 {
					connectionRatio = float64(activeConnections) / totalSystemEstimatedConns
				}

				// Cap the ratio at a reasonable value
				if connectionRatio > 0.9 {
					connectionRatio = 0.9 // Maximum 90% attribution to our process
				}
			} else {
				// If no active connections but we have network activity,
				// attribute a small percentage to our process
				if systemNetSent > 0 || systemNetReceived > 0 {
					connectionRatio = 0.05 // 5% attribution when we can't determine
				}
			}

			// Estimate process network I/O
			processNetSent := uint64(float64(systemNetSent) * connectionRatio)
			processNetReceived := uint64(float64(systemNetReceived) * connectionRatio)

			usage.ProcessNetSent = formatBytes(processNetSent) + "/s"
			usage.ProcessNetReceived = formatBytes(processNetReceived) + "/s"
		}
	}

	return usage, nil
}

// printSystemInfoTable prints system information in a table format
func printSystemInfoTable(sysInfo *SystemInfo) {
	cblog.Info("Printing system information table")

	fmt.Println()
	fmt.Println("-------------------------------- <System Information> ---------------------------------")

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Category", "Property", "Value"})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)

	// Set indentation prefix
	indentPrefix := "  " // 2 spaces

	// Add host information with indentation
	table.Append([]string{indentPrefix + "Host", "Hostname", sysInfo.Hostname})
	table.Append([]string{indentPrefix + "", "OS", fmt.Sprintf("%s %s", sysInfo.Platform, sysInfo.PlatformVersion)})
	table.Append([]string{indentPrefix + "", "Architecture", sysInfo.KernelArch})
	table.Append([]string{indentPrefix + "", "Kernel", sysInfo.KernelVersion})
	table.Append([]string{indentPrefix + "", "Uptime", sysInfo.Uptime})

	// Add CPU information with indentation
	table.Append([]string{indentPrefix + "CPU", "Model", sysInfo.CPUModel})
	table.Append([]string{indentPrefix + "", "Cores", fmt.Sprintf("%d physical, %d logical", sysInfo.PhysicalCores, sysInfo.LogicalCores)})
	table.Append([]string{indentPrefix + "", "Clock Speed", sysInfo.ClockSpeed})

	// Add memory information with indentation
	table.Append([]string{indentPrefix + "Memory", "Total Capacity", sysInfo.TotalMemory})
	table.Append([]string{indentPrefix + "", "Swap Capacity", sysInfo.SwapMemory})

	// Add disk information with indentation
	for i, part := range sysInfo.DiskPartitions {
		if i == 0 {
			table.Append([]string{indentPrefix + "Disk", fmt.Sprintf("Mount Point: %s", part.MountPoint), fmt.Sprintf("Total: %s", part.TotalSpace)})
		} else {
			table.Append([]string{indentPrefix + "", fmt.Sprintf("Mount Point: %s", part.MountPoint), fmt.Sprintf("Total: %s", part.TotalSpace)})
		}
	}

	table.Render()
	fmt.Println()
}

// printResourceUsageTable prints resource usage in a table format
func printResourceUsageTable(sysInfoTotalMemory string, usage *ResourceUsage) {
	cblog.Info("Printing resource usage table")

	fmt.Println("--------------------- <System Resource Usage and CB-Spider Usage> ---------------------")

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Resource", "System Total Usage", "CB-Spider Usage"})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)

	// Set indentation prefix
	indentPrefix := "  " // 2 spaces

	// Add CPU information with indentation
	table.Append([]string{
		indentPrefix + "CPU",
		usage.SystemCPUPercent,
		fmt.Sprintf("%s (%s)", usage.ProcessCPUPercent, usage.ProcessName),
	})

	// Add CPU core usage information from maps
	// Get sorted core keys for consistent ordering
	var coreKeys []string
	for coreKey := range usage.SystemCPUCorePercent {
		coreKeys = append(coreKeys, coreKey)
	}

	// With padded zeros in core keys, alphabetical sort will work correctly
	sort.Strings(coreKeys)

	// Display core usage in sorted order
	for _, coreKey := range coreKeys {
		sysCoreUsage := usage.SystemCPUCorePercent[coreKey]
		procCoreUsage := usage.ProcessCPUCorePercent[coreKey]

		table.Append([]string{
			indentPrefix + fmt.Sprintf("  %s", coreKey),
			sysCoreUsage,
			procCoreUsage,
		})
	}

	// Add memory information with indentation
	table.Append([]string{
		indentPrefix + "Memory (" + sysInfoTotalMemory + ")",
		fmt.Sprintf("%s (%s)", usage.SystemMemoryUsed, usage.SystemMemoryPercent),
		fmt.Sprintf("%s (%s)", usage.ProcessMemoryUsed, usage.ProcessMemoryPercent),
	})

	// Add disk I/O information with indentation
	table.Append([]string{
		indentPrefix + "Disk I/O",
		fmt.Sprintf("R: %s, W: %s", usage.SystemDiskRead, usage.SystemDiskWrite),
		fmt.Sprintf("R: %s, W: %s", usage.ProcessDiskRead, usage.ProcessDiskWrite),
	})

	// Add Network I/O information with indentation
	table.Append([]string{
		indentPrefix + "Network I/O",
		fmt.Sprintf("Sent: %s, Recv: %s", usage.SystemNetSent, usage.SystemNetReceived),
		fmt.Sprintf("Sent: %s, Recv: %s", usage.ProcessNetSent, usage.ProcessNetReceived),
	})

	table.Render()
}

// Format uptime to human-readable string
func formatUptime(uptime uint64) string {
	days := uptime / (60 * 60 * 24)
	hours := (uptime - (days * 60 * 60 * 24)) / (60 * 60)
	minutes := (uptime - (days * 60 * 60 * 24) - (hours * 60 * 60)) / 60

	result := ""
	if days > 0 {
		result += fmt.Sprintf("%d days, ", days)
	}
	if hours > 0 || days > 0 {
		result += fmt.Sprintf("%d hours, ", hours)
	}
	result += fmt.Sprintf("%d minutes", minutes)

	return result
}

// Format bytes to human-readable string
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
