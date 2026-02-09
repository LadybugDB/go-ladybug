// Package lbug provides a Go interface to Lbug graph database management system.
// The package is a wrapper around the C API of Lbug.
package lbug

// #include "lbug.h"
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"unsafe"
)

// SystemConfig represents the configuration of Lbug database system.
// BufferPoolSize is the size of the buffer pool in bytes.
// MaxNumThreads is the maximum number of threads that can be used by the database system.
// EnableCompression is a boolean flag to enable or disable compression.
// ReadOnly is a boolean flag to open the database in read-only mode.
// MaxDbSize is the maximum size of the database in bytes.
type SystemConfig struct {
	BufferPoolSize    uint64
	MaxNumThreads     uint64
	EnableCompression bool
	ReadOnly          bool
	MaxDbSize         uint64
}

// DefaultSystemConfig returns the default system configuration.
// The default system configuration is as follows:
// BufferPoolSize: 80% of the total system memory.
// MaxNumThreads: Number of CPU cores.
// EnableCompression: true.
// ReadOnly: false.
// MaxDbSize: 0 (unlimited).
func DefaultSystemConfig() SystemConfig {
	cSystemConfig := C.lbug_default_system_config()
	return SystemConfig{
		BufferPoolSize:    uint64(cSystemConfig.buffer_pool_size),
		MaxNumThreads:     uint64(cSystemConfig.max_num_threads),
		EnableCompression: bool(cSystemConfig.enable_compression),
		ReadOnly:          bool(cSystemConfig.read_only),
		MaxDbSize:         uint64(cSystemConfig.max_db_size),
	}
}

// toC converts the SystemConfig Go struct to the C struct.
func (config SystemConfig) toC() C.lbug_system_config {
	cSystemConfig := C.lbug_default_system_config()
	cSystemConfig.buffer_pool_size = C.uint64_t(config.BufferPoolSize)
	cSystemConfig.max_num_threads = C.uint64_t(config.MaxNumThreads)
	cSystemConfig.enable_compression = C.bool(config.EnableCompression)
	cSystemConfig.read_only = C.bool(config.ReadOnly)
	cSystemConfig.max_db_size = C.uint64_t(config.MaxDbSize)
	return cSystemConfig
}

// Database represents a Lbug database instance.
type Database struct {
	cDatabase C.lbug_database
	isClosed  bool
}

// OpenDatabase opens a Lbug database at the given path with the given system configuration.
func OpenDatabase(path string, systemConfig SystemConfig) (*Database, error) {
	db := &Database{}
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	cSystemConfig := systemConfig.toC()
	status := C.lbug_database_init(cPath, cSystemConfig, &db.cDatabase)
	if status != C.LbugSuccess {
		return db, fmt.Errorf("failed to open database with status %d", status)
	}
	return db, nil
}

// OpenInMemoryDatabase opens a Lbug database in in-memory mode with the given system configuration.
func OpenInMemoryDatabase(systemConfig SystemConfig) (*Database, error) {
	return OpenDatabase(":memory:", systemConfig)
}

// Close releases the underlying C resources for the database.
// MUST be called when done to prevent resource leaks.
// Use defer to ensure cleanup: defer db.Close()
func (db *Database) Close() {
	if db.isClosed {
		return
	}
	C.lbug_database_destroy(&db.cDatabase)
	db.isClosed = true
}
