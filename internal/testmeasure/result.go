package testmeasure

type Result struct {
	ElapsedNanoseconds int64 `json:"elapsed_nanoseconds"`
	CPUNanoseconds     int64 `json:"cpu_nanoseconds"`
	MaxRSSKB           int   `json:"max_rss_kb"`
}
