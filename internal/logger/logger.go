package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	logFile     *os.File
)

// InitLogger sets up the logging system
func InitLogger() error {
	today := time.Now().Format("2006-01-02")
	logDir := filepath.Join(os.Getenv("HOME"), "nfs_backup", "monitoring", "monitoring_util", today)
	
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory %s: %v", logDir, err)
	}
	
	logPath := filepath.Join(logDir, "info.log")
	
	// Open log file in append mode
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %v", logPath, err)
	}
	
	// Create multi-writer for both file and console
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	
	// Create loggers with timestamps
	InfoLogger = log.New(multiWriter, "[INFO] ", log.LstdFlags|log.Lshortfile)
	ErrorLogger = log.New(multiWriter, "[ERROR] ", log.LstdFlags|log.Lshortfile)
	
	InfoLogger.Printf("Logger initialized - log file: %s", logPath)
	return nil
}

// CloseLogger closes the log file
func CloseLogger() {
	if logFile != nil {
		InfoLogger.Println("Closing logger")
		logFile.Close()
	}
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	if InfoLogger != nil {
		InfoLogger.Printf(format, args...)
	} else {
		log.Printf("[INFO] "+format, args...)
	}
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Printf(format, args...)
	} else {
		log.Printf("[ERROR] "+format, args...)
	}
}

// LogRequest logs HTTP request details
func LogRequest(method, path, remoteAddr string, status int, duration time.Duration) {
	Info("HTTP %s %s from %s - Status: %d, Duration: %v", method, path, remoteAddr, status, duration)
	}

// LogError logs an error with context
func LogError(context string, err error) {
	Error("%s: %v", context, err)
}

// LogPanic logs a panic with context
func LogPanic(context string, recovered interface{}) {
	Error("PANIC in %s: %v", context, recovered)
}