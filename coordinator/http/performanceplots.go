package http

type performancePlotType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *HttpServer) performancePlotTypes() []performancePlotType {
	return []performancePlotType{
		{
			ID:   "cpu_usage",
			Name: "CPU Usage (total)",
		},
		{
			ID:   "system_memory",
			Name: "Memory Usage (total)",
		},
		{
			ID:   "network_buffers",
			Name: "Network Buffers",
		},
		{
			ID:   "num_threads",
			Name: "Thread count (monitored process)",
		},
		{
			ID:   "process_cpu_usage",
			Name: "CPU Usage (monitored process)",
		},
		{
			ID:   "disk_usage",
			Name: "Disk Usage (total)",
		},
		{
			ID:   "process_disk_usage",
			Name: "Disk Usage (process folder)",
		},
		{
			ID:   "flamegraph",
			Name: "Flame Graph",
		},
	}
}
