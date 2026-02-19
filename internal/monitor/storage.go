package monitor

import (
	"github.com/shirou/gopsutil/v4/disk"
)

type StorageMonitor struct {
	paths []string
}

func NewStorageMonitor(paths []string) *StorageMonitor {
	if len(paths) == 0 {
		paths = []string{"/"}
	}
	return &StorageMonitor{paths: paths}
}

func (m *StorageMonitor) Name() string {
	return "storage"
}

func (m *StorageMonitor) Collect() (any, error) {
	state := make(StorageState)

	for _, path := range m.paths {
		usage, err := disk.Usage(path)
		if err != nil {
			// Skip paths that are not accessible
			continue
		}

		state[path] = DiskState{
			UsedBytes:    usage.Used,
			TotalBytes:   usage.Total,
			UsagePercent: usage.UsedPercent,
		}
	}

	return state, nil
}
