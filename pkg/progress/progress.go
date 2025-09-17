package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// ProgressBar represents a progress bar
type ProgressBar struct {
	total     int
	current   int
	width     int
	showRate  bool
	showETA   bool
	startTime time.Time
	mu        sync.RWMutex
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int) *ProgressBar {
	return &ProgressBar{
		total:     total,
		current:   0,
		width:     50,
		showRate:  true,
		showETA:   true,
		startTime: time.Now(),
	}
}

// SetWidth sets the width of the progress bar
func (pb *ProgressBar) SetWidth(width int) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.width = width
}

// SetShowRate sets whether to show the rate
func (pb *ProgressBar) SetShowRate(show bool) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.showRate = show
}

// SetShowETA sets whether to show the ETA
func (pb *ProgressBar) SetShowETA(show bool) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.showETA = show
}

// Add increments the progress bar
func (pb *ProgressBar) Add(amount int) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.current += amount
	if pb.current > pb.total {
		pb.current = pb.total
	}
}

// Set sets the current progress
func (pb *ProgressBar) Set(current int) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.current = current
	if pb.current > pb.total {
		pb.current = pb.total
	}
}

// String returns the string representation of the progress bar
func (pb *ProgressBar) String() string {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	percentage := float64(pb.current) / float64(pb.total)
	filled := int(percentage * float64(pb.width))
	empty := pb.width - filled

	bar := strings.Repeat("=", filled) + strings.Repeat("-", empty)

	status := fmt.Sprintf("[%s] %d/%d (%.1f%%)", bar, pb.current, pb.total, percentage*100)

	if pb.showRate {
		elapsed := time.Since(pb.startTime)
		rate := float64(pb.current) / elapsed.Seconds()
		status += fmt.Sprintf(" %.1f/s", rate)
	}

	if pb.showETA && pb.current > 0 {
		elapsed := time.Since(pb.startTime)
		eta := time.Duration(float64(elapsed) * (float64(pb.total-pb.current) / float64(pb.current)))
		status += fmt.Sprintf(" ETA: %s", eta.Round(time.Second))
	}

	return status
}

// Write writes the progress bar to an io.Writer
func (pb *ProgressBar) Write(w io.Writer) {
	fmt.Fprintf(w, "\r%s", pb.String())
}

// Finish marks the progress bar as complete
func (pb *ProgressBar) Finish() {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.current = pb.total
}

// MultiProgressBar represents multiple progress bars
type MultiProgressBar struct {
	bars []*ProgressBar
	mu   sync.RWMutex
}

// NewMultiProgressBar creates a new multi-progress bar
func NewMultiProgressBar() *MultiProgressBar {
	return &MultiProgressBar{
		bars: make([]*ProgressBar, 0),
	}
}

// AddBar adds a progress bar
func (mpb *MultiProgressBar) AddBar(total int) *ProgressBar {
	mpb.mu.Lock()
	defer mpb.mu.Unlock()

	bar := NewProgressBar(total)
	mpb.bars = append(mpb.bars, bar)
	return bar
}

// Write writes all progress bars to an io.Writer
func (mpb *MultiProgressBar) Write(w io.Writer) {
	mpb.mu.RLock()
	defer mpb.mu.RUnlock()

	// Clear screen
	fmt.Fprint(w, "\033[2J\033[H")

	for i, bar := range mpb.bars {
		fmt.Fprintf(w, "Bar %d: %s\n", i+1, bar.String())
	}
}

// Spinner represents a spinner
type Spinner struct {
	chars   []string
	current int
	message string
	mu      sync.RWMutex
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		chars:   []string{"|", "/", "-", "\\"},
		current: 0,
		message: message,
	}
}

// Next advances the spinner
func (s *Spinner) Next() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = (s.current + 1) % len(s.chars)
}

// String returns the string representation of the spinner
func (s *Spinner) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fmt.Sprintf("%s %s", s.chars[s.current], s.message)
}

// Write writes the spinner to an io.Writer
func (s *Spinner) Write(w io.Writer) {
	fmt.Fprintf(w, "\r%s", s.String())
}

// ProgressTracker represents a progress tracker
type ProgressTracker struct {
	total     int
	current   int
	startTime time.Time
	mu        sync.RWMutex
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		total:     total,
		current:   0,
		startTime: time.Now(),
	}
}

// Add increments the progress
func (pt *ProgressTracker) Add(amount int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.current += amount
	if pt.current > pt.total {
		pt.current = pt.total
	}
}

// Set sets the current progress
func (pt *ProgressTracker) Set(current int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.current = current
	if pt.current > pt.total {
		pt.current = pt.total
	}
}

// GetProgress returns the current progress information
func (pt *ProgressTracker) GetProgress() ProgressInfo {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	percentage := float64(pt.current) / float64(pt.total)
	elapsed := time.Since(pt.startTime)

	var rate float64
	var eta time.Duration

	if pt.current > 0 {
		rate = float64(pt.current) / elapsed.Seconds()
		if rate > 0 {
			eta = time.Duration(float64(pt.total-pt.current) / rate * float64(time.Second))
		}
	}

	return ProgressInfo{
		Current:    pt.current,
		Total:      pt.total,
		Percentage: percentage,
		Elapsed:    elapsed,
		Rate:       rate,
		ETA:        eta,
	}
}

// ProgressInfo represents progress information
type ProgressInfo struct {
	Current    int
	Total      int
	Percentage float64
	Elapsed    time.Duration
	Rate       float64
	ETA        time.Duration
}

// String returns the string representation of the progress info
func (pi ProgressInfo) String() string {
	status := fmt.Sprintf("%d/%d (%.1f%%)", pi.Current, pi.Total, pi.Percentage*100)

	if pi.Rate > 0 {
		status += fmt.Sprintf(" %.1f/s", pi.Rate)
	}

	if pi.ETA > 0 {
		status += fmt.Sprintf(" ETA: %s", pi.ETA.Round(time.Second))
	}

	return status
}

// ProgressWriter represents a progress writer
type ProgressWriter struct {
	writer   io.Writer
	tracker  *ProgressTracker
	interval time.Duration
	mu       sync.Mutex
}

// NewProgressWriter creates a new progress writer
func NewProgressWriter(writer io.Writer, total int, interval time.Duration) *ProgressWriter {
	return &ProgressWriter{
		writer:   writer,
		tracker:  NewProgressTracker(total),
		interval: interval,
	}
}

// Write writes data and updates progress
func (pw *ProgressWriter) Write(p []byte) (n int, err error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	n, err = pw.writer.Write(p)
	pw.tracker.Add(n)

	// Update progress display
	progress := pw.tracker.GetProgress()
	fmt.Fprintf(pw.writer, "\r%s", progress.String())

	return n, err
}

// Close closes the progress writer
func (pw *ProgressWriter) Close() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	// Final progress update
	progress := pw.tracker.GetProgress()
	fmt.Fprintf(pw.writer, "\r%s\n", progress.String())

	return nil
}
