package handler

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

func BadRequestResponseBody(message string) map[string]any {
	return map[string]any{
		"message": fmt.Sprintf("Request has invalid format: `%s`", message),
	}
}

func UnprocessableEntityResponseBody(message string) map[string]any {
	return map[string]any{
		"message": fmt.Sprintf("Entity cannot be processed because of: `%s`", message),
	}
}

func InternalServerErrorResponseBody() map[string]any {
	return map[string]any{
		"message": "Internal server error",
	}
}

// MachineSpec represents information about the machine's hardware specifications
type MachineSpec struct {
	Ram struct {
		Total string `json:"total"`
		Free  string `json:"free"`
		Used  string `json:"used"`
	} `json:"ram"`

	Storage struct {
		Total string `json:"total"`
		Free  string `json:"free"`
		Used  string `json:"used"`
		INode struct {
			Total string `json:"total"`
			Free  string `json:"free"`
			Used  string `json:"used"`
		} `json:"inode"`
	} `json:"storage"`

	CPU struct {
		Cores int    `json:"cores"`
		Model string `json:"model"`

		Percentages []string `json:"percentages"`
	} `json:"cpu"`

	HostId         string `json:"host_id,omitempty"`
	HostName       string `json:"host_name,omitempty"`
	KernelVersion  string `json:"kernel_version,omitempty"`
	OSVersion      string `json:"os_version,omitempty"`
	Platform       string `json:"platform,omitempty"`
	PlatformFamily string `json:"platform_family,omitempty"`
	Uptime         int    `json:"uptime,omitempty"`
}

func getMachineSpecs() (MachineSpec, error) {
	specs := MachineSpec{}

	// Get CPU information using gopsutil
	cpuInfo, err := cpu.Info()
	if err != nil {
		return specs, err
	}
	if len(cpuInfo) == 0 {
		return specs, errors.New("no CPU information found")
	}

	specs.CPU.Cores = len(cpuInfo)
	specs.CPU.Model = cpuInfo[0].ModelName
	percentages, err := cpu.Percent(0, false)
	if err != nil {
		return specs, err
	}

	// Convert CPU percentages to strings
	for _, percentage := range percentages {
		specs.CPU.Percentages = append(specs.CPU.Percentages, strconv.FormatFloat(percentage, 'f', 2, 64)+"%")
	}

	// Get RAM information using gopsutil
	vMem, err := mem.VirtualMemory()
	if err != nil {
		return specs, err
	}
	specs.Ram.Total = strconv.FormatUint(vMem.Total/(1024*1024*1024), 10) + " GB"
	specs.Ram.Free = strconv.FormatUint(vMem.Free/(1024*1024*1024), 10) + " GB"
	specs.Ram.Used = strconv.FormatUint(vMem.Used/(1024*1024*1024), 10) + " GB"

	// Get Disk information using gopsutil
	diskUsage, err := disk.Usage("/")
	if err != nil {
		return specs, err
	}
	specs.Storage.Total = strconv.FormatUint(diskUsage.Total/(1024*1024*1024), 10) + " GB"
	specs.Storage.Free = strconv.FormatUint(diskUsage.Free/(1024*1024*1024), 10) + " GB"
	specs.Storage.Used = strconv.FormatUint(diskUsage.Used/(1024*1024*1024), 10) + " GB"
	specs.Storage.INode.Total = strconv.FormatUint(diskUsage.InodesTotal, 10)
	specs.Storage.INode.Free = strconv.FormatUint(diskUsage.InodesFree, 10)
	specs.Storage.INode.Used = strconv.FormatUint(diskUsage.InodesUsed, 10)

	// Get Host information using gopsutil
	hostInfo, err := host.Info()
	if err != nil {
		return specs, err
	}

	specs.HostId = hostInfo.HostID
	specs.HostName = hostInfo.Hostname
	specs.KernelVersion = hostInfo.KernelVersion
	specs.OSVersion = hostInfo.OS
	specs.Platform = hostInfo.Platform
	specs.PlatformFamily = hostInfo.PlatformFamily

	// Convert uptime to integer hours
	uptimeHours := int(hostInfo.Uptime / 3600)
	specs.Uptime = uptimeHours

	return specs, nil
}
