package utils

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarningLevel
	ErrorLevel
	FatalLevel
)

// LogRotationConfig holds configuration for log rotation
type LogRotationConfig struct {
	MaxSizeMB     int           // Maximum size of a log file in MB before rotation (0 = no size limit)
	MaxAgeDays    int           // Maximum age of log files in days before deletion (0 = keep all)
	MaxBackups    int           // Maximum number of old log files to keep (0 = keep all)
	Compress      bool          // Whether to compress rotated log files
	RotateOnStart bool          // Whether to rotate logs on startup
	CheckInterval time.Duration // How often to check for rotation (default: every write)
}

// DefaultLogRotationConfig returns default rotation settings
func DefaultLogRotationConfig() LogRotationConfig {
	return LogRotationConfig{
		MaxSizeMB:     100,  // 100MB max file size
		MaxAgeDays:    30,   // Keep logs for 30 days
		MaxBackups:    10,   // Keep 10 old files
		Compress:      true, // Compress old logs
		RotateOnStart: false,
		CheckInterval: 0, // Check on every write
	}
}

type Logger struct {
	*log.Logger
	file           *os.File
	dir            string
	uid            int
	gid            int
	mu             sync.RWMutex
	minLevel       LogLevel
	env            string
	rotationConfig LogRotationConfig
	lastCheck      time.Time
}

var globalLogger *Logger

// getLogLevelForEnv determines the minimum log level based on environment
func getLogLevelForEnv(env string) LogLevel {
	env = strings.ToLower(env)
	switch env {
	case "production", "prod":
		return WarningLevel // Only WARNING and ERROR in production
	case "staging", "stage":
		return InfoLevel // INFO, WARNING, and ERROR in staging
	case "development", "dev":
		return DebugLevel // All logs in development
	default:
		return DebugLevel // Default to debug for unknown environments
	}
}

// getLogOutput determines the output writer based on environment
func getLogOutput(file *os.File, env string) io.Writer {
	env = strings.ToLower(env)
	switch env {
	case "production", "prod":
		// Production: only write to file
		return file
	case "staging", "stage", "development", "dev":
		// Staging and development: write to both file and stdout
		return io.MultiWriter(file, os.Stdout)
	default:
		// Default: write to both
		return io.MultiWriter(file, os.Stdout)
	}
}

// NewLogger creates a new logger instance with configurable UID/GID and environment-based logging
func NewLogger(logDir string, uid, gid int, env string) (*Logger, error) {
	return NewLoggerWithRotation(logDir, uid, gid, env, DefaultLogRotationConfig())
}

// NewLoggerWithRotation creates a new logger instance with custom rotation configuration
func NewLoggerWithRotation(logDir string, uid, gid int, env string, rotationConfig LogRotationConfig) (*Logger, error) {
	// Convert to absolute path to avoid relative path issues
	absLogDir, err := filepath.Abs(logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for log directory: %w", err)
	}

	// Create log directory with proper permissions
	if err := CreateDirectoryWithPermissions(absLogDir, uid, gid); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	filename := filepath.Join(absLogDir, time.Now().Format("2006-01-02")+".log")

	// Create log file with appropriate permissions for the uploader user
	file, err := CreateFileWithPermissions(filename, uid, gid, 0664)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", filename, err)
	}

	// Get environment-specific configuration
	minLevel := getLogLevelForEnv(env)
	output := getLogOutput(file, env)

	logger := &Logger{
		Logger:         log.New(output, "", log.LstdFlags|log.Lshortfile),
		file:           file,
		dir:            absLogDir,
		uid:            uid,
		gid:            gid,
		minLevel:       minLevel,
		env:            env,
		rotationConfig: rotationConfig,
		lastCheck:      time.Now(),
	}

	// Set this as the global logger
	globalLogger = logger

	// Replace the default Go logger with our custom logger
	log.SetOutput(output)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Clean up old log files based on retention policy
	logger.cleanupOldLogs()

	return logger, nil
}

// SetupGlobalLogger sets up the global logger with default rotation configuration
func SetupGlobalLogger(logDir string, uid, gid int, env string) error {
	logger, err := NewLogger(logDir, uid, gid, env)
	if err != nil {
		return err
	}

	globalLogger = logger
	return nil
}

// SetupGlobalLoggerWithRotation sets up the global logger with custom rotation configuration
func SetupGlobalLoggerWithRotation(logDir string, uid, gid int, env string, rotationConfig LogRotationConfig) error {
	logger, err := NewLoggerWithRotation(logDir, uid, gid, env, rotationConfig)
	if err != nil {
		return err
	}

	globalLogger = logger
	return nil
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	return globalLogger
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// checkAndRotateLog checks if we need to rotate the log file (new day or size limit)
func (l *Logger) checkAndRotateLog() {
	// Check if we should skip this check based on interval
	if l.rotationConfig.CheckInterval > 0 {
		if time.Since(l.lastCheck) < l.rotationConfig.CheckInterval {
			return
		}
		l.lastCheck = time.Now()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	stat, err := l.file.Stat()
	if err != nil {
		return
	}

	needsRotation := false
	rotationReason := ""

	// Check for daily rotation (new day)
	currentDate := time.Now().Format("2006-01-02")
	currentFilename := filepath.Base(stat.Name())
	expectedFilename := currentDate + ".log"

	if currentFilename != expectedFilename {
		needsRotation = true
		rotationReason = "daily rotation"
	}

	// Check for size-based rotation
	if l.rotationConfig.MaxSizeMB > 0 {
		maxSize := int64(l.rotationConfig.MaxSizeMB) * 1024 * 1024 // Convert MB to bytes
		if stat.Size() >= maxSize {
			needsRotation = true
			rotationReason = "size limit exceeded"
		}
	}

	if !needsRotation {
		return
	}

	// Perform rotation
	if err := l.rotateLog(rotationReason); err != nil {
		fmt.Printf("Error: failed to rotate log: %v\n", err)
	}
}

// rotateLog performs the actual log rotation
func (l *Logger) rotateLog(reason string) error {
	// Close current file
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close current log file: %w", err)
	}

	// Get current file info for renaming
	oldFilename := l.file.Name()

	// Create backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	ext := filepath.Ext(oldFilename)
	nameWithoutExt := strings.TrimSuffix(filepath.Base(oldFilename), ext)
	backupFilename := filepath.Join(l.dir, fmt.Sprintf("%s_%s%s", nameWithoutExt, timestamp, ext))

	// Rename current file to backup
	if err := os.Rename(oldFilename, backupFilename); err != nil {
		fmt.Printf("Warning: failed to rename log file: %v\n", err)
	} else {
		// Compress the rotated file if configured
		if l.rotationConfig.Compress {
			go l.compressLogFile(backupFilename)
		}
	}

	// Create new log file
	currentDate := time.Now().Format("2006-01-02")
	newFilename := filepath.Join(l.dir, currentDate+".log")

	file, err := CreateFileWithPermissions(newFilename, l.uid, l.gid, 0664)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	l.file = file

	// Update the logger output
	output := getLogOutput(file, l.env)
	l.Logger.SetOutput(output)
	log.SetOutput(output)

	// Clean up old logs
	go l.cleanupOldLogs()

	fmt.Printf("Log rotated: %s (reason: %s)\n", newFilename, reason)
	return nil
}

// compressLogFile compresses a log file using gzip
func (l *Logger) compressLogFile(filename string) {
	// Open the source file
	srcFile, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Warning: failed to open file for compression: %v\n", err)
		return
	}
	defer srcFile.Close()

	// Create compressed file
	gzFilename := filename + ".gz"
	gzFile, err := os.Create(gzFilename)
	if err != nil {
		fmt.Printf("Warning: failed to create compressed file: %v\n", err)
		return
	}
	defer gzFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(gzFile)
	defer gzWriter.Close()

	// Copy and compress
	if _, err := io.Copy(gzWriter, srcFile); err != nil {
		fmt.Printf("Warning: failed to compress file: %v\n", err)
		os.Remove(gzFilename) // Remove incomplete compressed file
		return
	}

	// Remove original file after successful compression
	if err := os.Remove(filename); err != nil {
		fmt.Printf("Warning: failed to remove original log file after compression: %v\n", err)
	}
}

// cleanupOldLogs removes old log files based on retention policy
func (l *Logger) cleanupOldLogs() {
	files, err := os.ReadDir(l.dir)
	if err != nil {
		fmt.Printf("Warning: failed to read log directory: %v\n", err)
		return
	}

	type logFileInfo struct {
		name    string
		modTime time.Time
		size    int64
	}

	var logFiles []logFileInfo

	// Collect all log files (both .log and .log.gz)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasSuffix(name, ".log") && !strings.HasSuffix(name, ".log.gz") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		// Skip the current log file
		if filepath.Join(l.dir, name) == l.file.Name() {
			continue
		}

		logFiles = append(logFiles, logFileInfo{
			name:    name,
			modTime: info.ModTime(),
			size:    info.Size(),
		})
	}

	// Sort by modification time (oldest first)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].modTime.Before(logFiles[j].modTime)
	})

	now := time.Now()

	// Remove files based on retention policies
	for _, file := range logFiles {
		shouldRemove := false
		reason := ""

		// Check age-based retention
		if l.rotationConfig.MaxAgeDays > 0 {
			age := now.Sub(file.modTime)
			if age > time.Duration(l.rotationConfig.MaxAgeDays)*24*time.Hour {
				shouldRemove = true
				reason = "exceeded max age"
			}
		}

		// Check count-based retention
		if l.rotationConfig.MaxBackups > 0 && len(logFiles) > l.rotationConfig.MaxBackups {
			// Remove oldest files first
			shouldRemove = true
			reason = "exceeded max backups"
		}

		if shouldRemove {
			fullPath := filepath.Join(l.dir, file.name)
			if err := os.Remove(fullPath); err != nil {
				fmt.Printf("Warning: failed to remove old log file %s: %v\n", file.name, err)
			} else {
				fmt.Printf("Removed old log file: %s (reason: %s)\n", file.name, reason)
			}
		}
	}
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.checkAndRotateLog()
	l.Logger.Printf(format, v...)
}

func (l *Logger) Print(v ...interface{}) {
	l.checkAndRotateLog()
	l.Logger.Print(v...)
}

func (l *Logger) Println(v ...interface{}) {
	l.checkAndRotateLog()
	l.Logger.Println(v...)
}

// shouldLog checks if a message at the given level should be logged
func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.minLevel
}

func (l *Logger) Info(format string, v ...interface{}) {
	if !l.shouldLog(InfoLevel) {
		return
	}
	l.Printf("[INFO] "+format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	if !l.shouldLog(ErrorLevel) {
		return
	}
	l.Printf("[ERROR] "+format, v...)
}

func (l *Logger) Warning(format string, v ...interface{}) {
	if !l.shouldLog(WarningLevel) {
		return
	}
	l.Printf("[WARNING] "+format, v...)
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if !l.shouldLog(DebugLevel) {
		return
	}
	l.Printf("[DEBUG] "+format, v...)
}

func (l *Logger) Fatal(format string, v ...interface{}) {
	if !l.shouldLog(FatalLevel) {
		return
	}
	l.Printf("[FATAL] "+format, v...)
	os.Exit(1)
}

// Global convenience functions that use the global logger
func Info(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Info(format, v...)
	} else {
		fmt.Printf("[INFO] "+format+"\n", v...)
	}
}

func Error(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Error(format, v...)
	} else {
		fmt.Printf("[ERROR] "+format+"\n", v...)
	}
}

func Warning(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Warning(format, v...)
	} else {
		fmt.Printf("[WARNING] "+format+"\n", v...)
	}
}

func Debug(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Debug(format, v...)
	} else {
		fmt.Printf("[DEBUG] "+format+"\n", v...)
	}
}

func Fatal(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Fatal(format, v...)
	} else {
		fmt.Printf("[FATAL] "+format+"\n", v...)
		os.Exit(1)
	}
}
