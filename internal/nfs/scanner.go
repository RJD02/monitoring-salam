package nfs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"salam-monitoring/internal/logger"
)

// LogEntry represents a log entry from NFS monitoring
type LogEntry struct {
	Source    string    `json:"source"`
	Date      string    `json:"date"`
	Workflow  string    `json:"workflow"`
	LogType   string    `json:"log_type"`
	Content   string    `json:"content"`
	HasErrors bool      `json:"has_errors"`
	FilePath  string    `json:"file_path"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"mod_time"`
}

// WorkflowSummary represents a summary of workflow logs
type WorkflowSummary struct {
	Source    string      `json:"source"`
	Date      string      `json:"date"`
	Workflow  string      `json:"workflow"`
	Logs      []*LogEntry `json:"logs"`
	HasErrors bool        `json:"has_errors"`
	Status    string      `json:"status"`
}

// Scanner handles NFS log scanning operations
type Scanner struct {
	nfsRoot string
}

// NewScanner creates a new NFS log scanner
func NewScanner(nfsRoot string) *Scanner {
	logger.Info("Creating NFS scanner for root: %s", nfsRoot)
	return &Scanner{
		nfsRoot: nfsRoot,
	}
}

// ScanTodaysLogs scans today's logs from all sources
func (s *Scanner) ScanTodaysLogs() ([]*WorkflowSummary, error) {
	today := time.Now().Format("2006-01-02")
	logger.Info("Scanning today's logs for date: %s", today)
	return s.ScanLogsForDate(today)
}

// ScanLogsForDate scans logs for a specific date
func (s *Scanner) ScanLogsForDate(date string) ([]*WorkflowSummary, error) {
	logger.Info("Scanning logs for date: %s in NFS root: %s", date, s.nfsRoot)

	// Scan all source directories
	var summaries []*WorkflowSummary
	sources, err := s.getSourceDirectories()
	if err != nil {
		return nil, fmt.Errorf("failed to get source directories: %w", err)
	}

	for _, source := range sources {
		sourceSummaries, err := s.scanSourceForDate(source, date)
		if err != nil {
			// Log error but continue with other sources
			logger.LogError(fmt.Sprintf("Failed to scan source %s for date %s", source, date), err)
			continue
		}
		summaries = append(summaries, sourceSummaries...)
	}

	// Sort summaries by source and workflow name
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Source != summaries[j].Source {
			return summaries[i].Source < summaries[j].Source
		}
		return summaries[i].Workflow < summaries[j].Workflow
	})

	logger.Info("Found %d workflow summaries for date %s", len(summaries), date)
	return summaries, nil
}

// getSourceDirectories returns all source directories under NFS root
func (s *Scanner) getSourceDirectories() ([]string, error) {
	entries, err := os.ReadDir(s.nfsRoot)
	if err != nil {
		return nil, err
	}

	var sources []string
	for _, entry := range entries {
		if entry.IsDir() {
			sources = append(sources, entry.Name())
		}
	}
	return sources, nil
}

// scanSourceForDate scans a specific source directory for a specific date
func (s *Scanner) scanSourceForDate(source, date string) ([]*WorkflowSummary, error) {
	datePath := filepath.Join(s.nfsRoot, source, date)
	var summaries []*WorkflowSummary

	// Date directory doesn't exist, return empty result
	if _, err := os.Stat(datePath); os.IsNotExist(err) {
		return summaries, nil
	}

	// Get all workflow directories for this date
	workflows, err := s.getWorkflowDirectories(datePath)
	if err != nil {
		return nil, err
	}

	for _, workflow := range workflows {
		summary, err := s.scanWorkflow(source, date, workflow)
		if err != nil {
			logger.LogError(fmt.Sprintf("Failed to scan workflow %s", workflow), err)
			continue
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

// getWorkflowDirectories returns all workflow directories under a date path
func (s *Scanner) getWorkflowDirectories(datePath string) ([]string, error) {
	entries, err := os.ReadDir(datePath)
	if err != nil {
		return nil, err
	}

	var workflows []string
	for _, entry := range entries {
		if entry.IsDir() {
			workflows = append(workflows, entry.Name())
		}
	}
	return workflows, nil
}

// scanWorkflow scans a specific workflow directory for logs
func (s *Scanner) scanWorkflow(source, date, workflow string) (*WorkflowSummary, error) {
	workflowPath := filepath.Join(s.nfsRoot, source, date, workflow)

	summary := &WorkflowSummary{
		Source:   source,
		Date:     date,
		Workflow: workflow,
		Logs:     []*LogEntry{},
		Status:   "Unknown",
	}

	// Scan for log files
	logTypes := []string{"info.log", "error.log", "run.log"}
	for _, logType := range logTypes {
		logPath := filepath.Join(workflowPath, logType)
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			continue // File doesn't exist, skip
		}

		logEntry, err := s.scanLogFile(source, date, workflow, logType, logPath)
		if err != nil {
			logger.LogError(fmt.Sprintf("Failed to scan log file %s", logPath), err)
			continue
		}

		summary.Logs = append(summary.Logs, logEntry)
		if logEntry.HasErrors {
			summary.HasErrors = true
		}
	}

	// Determine workflow status
	summary.Status = s.determineWorkflowStatus(summary)
	return summary, nil
}

// scanLogFile scans a specific log file
func (s *Scanner) scanLogFile(source, date, workflow, logType, filePath string) (*LogEntry, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Read file content for error detection
	hasErrors, err := s.detectErrors(filePath, logType)
	if err != nil {
		return nil, err
	}

	entry := &LogEntry{
		Source:    source,
		Date:      date,
		Workflow:  workflow,
		LogType:   logType,
		HasErrors: hasErrors,
		FilePath:  filePath,
		Size:      stat.Size(),
		ModTime:   stat.ModTime(),
	}
	return entry, nil
}

// detectErrors scans a log file for error indicators
func (s *Scanner) detectErrors(filePath, logType string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	errorPatterns := []string{
		"ERROR",
		"FATAL",
		"Exception",
		"Failed",
		"failure",
		"FAILED",
		"error:",
		"Error:",
	}

	// For error.log files, any content indicates errors
	if logType == "error.log" {
		// Check if file has any content
		scanner.Scan()
		return len(strings.TrimSpace(scanner.Text())) > 0, scanner.Err()
	}

	// For other logs, scan for error patterns
	for scanner.Scan() {
		line := scanner.Text()
		for _, pattern := range errorPatterns {
			if strings.Contains(line, pattern) {
				return true, nil
			}
		}
	}

	return false, scanner.Err()
}

// determineWorkflowStatus determines the overall workflow status
func (s *Scanner) determineWorkflowStatus(summary *WorkflowSummary) string {
	if summary.HasErrors {
		return "Failed"
	}

	// Check if we have any logs
	if len(summary.Logs) == 0 {
		return "No Logs"
	}

	// Check for run.log to determine if workflow completed
	hasRunLog := false
	for _, log := range summary.Logs {
		if log.LogType == "run.log" {
			hasRunLog = true
			break
		}
	}

	if hasRunLog && !summary.HasErrors {
		return "Completed"
	}

	return "In Progress"
}

// GetLogContent reads the content of a specific log file
func (s *Scanner) GetLogContent(filePath string, maxLines int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() && (maxLines == 0 || lineCount < maxLines) {
		lines = append(lines, scanner.Text())
		lineCount++
	}

	return lines, scanner.Err()
}

// GetLogTail reads the last N lines of a log file
func (s *Scanner) GetLogTail(filePath string, lines int) ([]string, error) {
	// In production, you might want to use more efficient tail implementation
	// For simplicity, we'll read the whole file and return the last N lines
	allLines, err := s.GetLogContent(filePath, 0)
	if err != nil {
		return nil, err
	}

	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}

	return allLines[start:], nil
}

// SearchLogs searches for a keyword across all logs for today
func (s *Scanner) SearchLogs(keyword string) ([]*LogEntry, error) {
	summaries, err := s.ScanTodaysLogs()
	if err != nil {
		return nil, err
	}

	var results []*LogEntry
	for _, summary := range summaries {
		for _, logEntry := range summary.Logs {
			content, err := s.GetLogContent(logEntry.FilePath, 1000) // Search first 1000 lines
			if err != nil {
				continue
			}

			for _, line := range content {
				if strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
					// Clone the log entry with matching content
					result := *logEntry
					result.Content = line
					results = append(results, &result)
					break // Only add once per file
				}
			}
		}
	}

	return results, nil
}
