package app

// Result is the outcome of converting a single file within a batch.
type Result int

const (
	// ResultProcessing indicates a file conversion has started.
	ResultProcessing Result = iota
	// ResultSucceeded indicates a file was converted successfully.
	ResultSucceeded
	// ResultSkipped indicates the destination already existed.
	ResultSkipped
	// ResultFailed indicates the conversion failed.
	ResultFailed
)

// Progress receives batch-level and per-file conversion events. Methods may be
// called from worker goroutines; UI implementations must marshal updates to the
// UI thread (for example via fyne.Do).
type Progress interface {
	Start(total int)
	File(path string, result Result, err error) // err is non-nil only for ResultFailed
	Done()
}
