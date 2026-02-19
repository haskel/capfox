package monitor

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/process"
)

type ProcessMonitor struct {
	prevCtxSwitches uint64
	prevTime        time.Time
	mu              sync.Mutex
}

func NewProcessMonitor() *ProcessMonitor {
	return &ProcessMonitor{
		prevTime: time.Now(),
	}
}

func (m *ProcessMonitor) Name() string {
	return "process"
}

func (m *ProcessMonitor) Collect() (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get process count
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}
	processCount := len(procs)

	// Count threads
	var threadCount int
	for _, p := range procs {
		threads, err := p.NumThreads()
		if err == nil {
			threadCount += int(threads)
		}
	}

	// Get context switches
	cpuTimes, err := cpu.Times(false)
	if err != nil {
		return nil, err
	}

	var ctxSwitchesPerSec int64
	now := time.Now()

	if len(cpuTimes) > 0 {
		// Use Idle + System as approximation for context switches
		// This is a simplification - real ctx switches would need /proc/stat on Linux
		currentCtx := uint64(cpuTimes[0].Idle + cpuTimes[0].System)

		if m.prevCtxSwitches > 0 {
			elapsed := now.Sub(m.prevTime).Seconds()
			if elapsed > 0 {
				ctxSwitchesPerSec = int64(float64(currentCtx-m.prevCtxSwitches) / elapsed)
			}
		}

		m.prevCtxSwitches = currentCtx
	}

	m.prevTime = now

	return &ProcessState{
		Processes:             processCount,
		Threads:               threadCount,
		ContextSwitchesPerSec: ctxSwitchesPerSec,
	}, nil
}
