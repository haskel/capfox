package monitor

// GPUMonitor collects NVIDIA GPU metrics.
// Graceful degradation: if NVML is not available, returns empty slice.
type GPUMonitor struct {
	available bool
}

func NewGPUMonitor() *GPUMonitor {
	m := &GPUMonitor{
		available: false,
	}

	// Try to initialize NVML
	// For now, we'll implement a stub that returns empty data
	// Real implementation would use github.com/NVIDIA/go-nvml/pkg/nvml
	m.available = initNVML()

	return m
}

func (m *GPUMonitor) Name() string {
	return "gpu"
}

func (m *GPUMonitor) Collect() (any, error) {
	if !m.available {
		return []GPUState{}, nil
	}

	return collectGPUMetrics()
}

func (m *GPUMonitor) Available() bool {
	return m.available
}

func (m *GPUMonitor) Close() error {
	if m.available {
		return shutdownNVML()
	}
	return nil
}

// Stub implementations - will be replaced with real NVML calls
// when github.com/NVIDIA/go-nvml is added

func initNVML() bool {
	// TODO: Implement real NVML initialization
	// ret := nvml.Init()
	// return ret == nvml.SUCCESS
	return false
}

func shutdownNVML() error {
	// TODO: Implement real NVML shutdown
	// nvml.Shutdown()
	return nil
}

func collectGPUMetrics() ([]GPUState, error) {
	// TODO: Implement real GPU metrics collection
	// count, _ := nvml.DeviceGetCount()
	// for i := 0; i < count; i++ {
	//     device, _ := nvml.DeviceGetHandleByIndex(i)
	//     ...
	// }
	return []GPUState{}, nil
}
