package gdb

import (
	_ "embed"
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed firmwares/firmwares.yaml
var firmwaresYAML []byte

// Firmware represents a CC3200 firmware version with function addresses.
type Firmware struct {
	// Version is the firmware version identifier (e.g., "0x355")
	Version string `yaml:"version"`

	// Name is the human-readable firmware name
	Name string `yaml:"name"`

	// Description provides details about this firmware variant
	Description string `yaml:"description"`

	// Verified indicates whether this firmware has been tested
	Verified bool `yaml:"verified"`

	// Functions contains memory addresses of TI SimpleLink API functions
	Functions FirmwareFunctions `yaml:"functions"`

	// Memory contains memory locations for GDB operations
	Memory FirmwareMemory `yaml:"memory"`

	// Notes contains additional information about this firmware
	Notes string `yaml:"notes"`
}

// FirmwareFunctions holds memory addresses of TI SimpleLink API functions.
type FirmwareFunctions struct {
	// sl_FsOpen: Open/create file in device flash
	SlFsOpen int64 `yaml:"sl_FsOpen"`

	// sl_FsRead: Read from file
	SlFsRead int64 `yaml:"sl_FsRead"`

	// sl_FsWrite: Write to file
	SlFsWrite int64 `yaml:"sl_FsWrite"`

	// sl_FsClose: Close file
	SlFsClose int64 `yaml:"sl_FsClose"`

	// sl_FsDel: Delete file
	SlFsDel int64 `yaml:"sl_FsDel"`

	// sl_FsGetInfo: Get file information
	SlFsGetInfo int64 `yaml:"sl_FsGetInfo"`

	// UartLog: UART logging function (for log capture)
	UartLog int64 `yaml:"uart_log"`

	// Signatures contains function signatures for firmware detection.
	// Format: map[function_name][2]uint32 where each entry is two 32-bit words
	// representing the first 8 bytes at that function address.
	// This is optional for backward compatibility with firmwares that don't have signatures yet.
	Signatures map[string][2]uint32 `yaml:"signatures,omitempty"`
}

// FirmwareMemory holds memory locations for GDB operations.
type FirmwareMemory struct {
	// WorkBuffer: Temporary storage for data (certificates, file content)
	WorkBuffer int64 `yaml:"work_buffer"`

	// FileHandlePtr: Location to store sl_FsOpen return value
	FileHandlePtr int64 `yaml:"file_handle_ptr"`

	// FilenamePtr: Location to store filename string
	FilenamePtr int64 `yaml:"filename_ptr"`

	// TokenPtr: Location to store file token
	TokenPtr int64 `yaml:"token_ptr"`

	// StackBase: Safe location for stack-based operations
	StackBase int64 `yaml:"stack_base"`
}

// FirmwareDB holds all known firmware versions.
type FirmwareDB struct {
	// Firmwares is a slice of all known firmware versions
	Firmwares []*Firmware

	// index maps version strings to firmware entries for fast lookup
	index map[string]*Firmware

	// mu protects the index during lazy initialization
	mu sync.RWMutex
}

// firmwareDBContainer is for YAML unmarshaling
type firmwareDBContainer struct {
	Firmwares []*Firmware `yaml:"firmwares"`
}

var (
	// globalFirmwareDB is the singleton firmware database
	globalFirmwareDB *FirmwareDB
	// globalFirmwareOnce ensures we only load the database once
	globalFirmwareOnce sync.Once
	// globalFirmwareErr stores any error from loading
	globalFirmwareErr error
)

// LoadFirmwares loads the embedded firmware catalog and returns a FirmwareDB.
// This function is safe to call multiple times; the database is loaded only once.
func LoadFirmwares() (*FirmwareDB, error) {
	globalFirmwareOnce.Do(func() {
		globalFirmwareDB, globalFirmwareErr = loadFirmwaresInternal()
	})
	return globalFirmwareDB, globalFirmwareErr
}

// loadFirmwaresInternal does the actual loading of the firmware catalog.
func loadFirmwaresInternal() (*FirmwareDB, error) {
	var container firmwareDBContainer
	if err := yaml.Unmarshal(firmwaresYAML, &container); err != nil {
		return nil, fmt.Errorf("failed to parse firmwares.yaml: %w", err)
	}

	db := &FirmwareDB{
		Firmwares: container.Firmwares,
		index:     make(map[string]*Firmware),
	}

	// Build index
	for _, fw := range db.Firmwares {
		db.index[fw.Version] = fw
	}

	return db, nil
}

// Get retrieves a firmware by version string.
// Returns nil, false if the version is not found.
func (db *FirmwareDB) Get(version string) (*Firmware, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	fw, ok := db.index[version]
	return fw, ok
}

// List returns all known firmware versions.
func (db *FirmwareDB) List() []*Firmware {
	return db.Firmwares
}

// Versions returns all version strings in the catalog.
func (db *FirmwareDB) Versions() []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	versions := make([]string, 0, len(db.index))
	for version := range db.index {
		versions = append(versions, version)
	}
	return versions
}

// Count returns the number of firmware versions in the catalog.
func (db *FirmwareDB) Count() int {
	return len(db.Firmwares)
}

// GetVerified returns only verified firmware versions.
func (db *FirmwareDB) GetVerified() []*Firmware {
	verified := make([]*Firmware, 0)
	for _, fw := range db.Firmwares {
		if fw.Verified {
			verified = append(verified, fw)
		}
	}
	return verified
}

// String returns a human-readable representation of the firmware.
func (f *Firmware) String() string {
	verifiedStr := ""
	if f.Verified {
		verifiedStr = " (verified)"
	}
	return fmt.Sprintf("%s - %s%s", f.Version, f.Name, verifiedStr)
}

// FormatAddresses returns a formatted string of all function addresses.
func (f *Firmware) FormatAddresses() string {
	return fmt.Sprintf(`Function Addresses:
  sl_FsOpen:    0x%08x
  sl_FsRead:    0x%08x
  sl_FsWrite:   0x%08x
  sl_FsClose:   0x%08x
  sl_FsDel:     0x%08x
  sl_FsGetInfo: 0x%08x
  uart_log:     0x%08x

Memory Locations:
  work_buffer:     0x%08x
  file_handle_ptr: 0x%08x
  filename_ptr:    0x%08x
  token_ptr:       0x%08x
  stack_base:      0x%08x`,
		f.Functions.SlFsOpen,
		f.Functions.SlFsRead,
		f.Functions.SlFsWrite,
		f.Functions.SlFsClose,
		f.Functions.SlFsDel,
		f.Functions.SlFsGetInfo,
		f.Functions.UartLog,
		f.Memory.WorkBuffer,
		f.Memory.FileHandlePtr,
		f.Memory.FilenamePtr,
		f.Memory.TokenPtr,
		f.Memory.StackBase,
	)
}

// HandleUnknownFirmware creates a helpful error for unsupported firmware versions.
func HandleUnknownFirmware(version string) error {
	db, err := LoadFirmwares()
	if err != nil {
		return fmt.Errorf("firmware version %s is unknown and cannot load firmware catalog: %w", version, err)
	}

	return &FirmwareUnsupportedError{
		Version:   version,
		Available: db.Versions(),
	}
}
