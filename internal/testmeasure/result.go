package testmeasure

type Result struct {
	ElapsedNanoseconds int64  `json:"elapsed_nanoseconds"`
	CPUNanoseconds     int64  `json:"cpu_nanoseconds"`
	MaxRSSKB           int    `json:"max_rss_kb"`
	PeakMemoryBytes    uint64 `json:"peak_memory_bytes"`
	MemoryMetric       string `json:"memory_metric"`
}
