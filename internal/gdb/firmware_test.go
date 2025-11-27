package gdb

import (
	"strings"
	"testing"
)

func TestLoadFirmwares(t *testing.T) {
	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	if db == nil {
		t.Fatal("expected non-nil FirmwareDB")
	}

	if db.Firmwares == nil {
		t.Fatal("expected non-nil Firmwares slice")
	}

	if len(db.Firmwares) == 0 {
		t.Fatal("expected at least one firmware in catalog")
	}

	// Should be able to call LoadFirmwares multiple times (singleton pattern)
	db2, err2 := LoadFirmwares()
	if err2 != nil {
		t.Fatalf("second LoadFirmwares failed: %v", err2)
	}

	// Should return the same instance
	if db != db2 {
		t.Error("expected LoadFirmwares to return same instance (singleton)")
	}
}

func TestFirmwareDB_Get(t *testing.T) {
	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	// Test getting a known firmware (0x355 should exist)
	fw, ok := db.Get("0x355")
	if !ok {
		t.Error("expected to find firmware version 0x355")
	}

	if fw == nil {
		t.Fatal("expected non-nil firmware")
	}

	if fw.Version != "0x355" {
		t.Errorf("expected version '0x355', got %s", fw.Version)
	}

	// Test getting a non-existent firmware
	_, ok = db.Get("0xnonexistent")
	if ok {
		t.Error("expected not to find nonexistent firmware version")
	}
}

func TestFirmwareDB_List(t *testing.T) {
	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	list := db.List()
	if list == nil {
		t.Fatal("expected non-nil list")
	}

	if len(list) == 0 {
		t.Fatal("expected at least one firmware in list")
	}

	// List should match Firmwares field
	if len(list) != len(db.Firmwares) {
		t.Errorf("expected list length %d, got %d", len(db.Firmwares), len(list))
	}
}

func TestFirmwareDB_Versions(t *testing.T) {
	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	versions := db.Versions()
	if versions == nil {
		t.Fatal("expected non-nil versions slice")
	}

	if len(versions) == 0 {
		t.Fatal("expected at least one version")
	}

	// All versions should be non-empty strings
	for _, v := range versions {
		if v == "" {
			t.Error("found empty version string")
		}
	}

	// Should contain 0x355
	found := false
	for _, v := range versions {
		if v == "0x355" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected versions to contain '0x355'")
	}
}

func TestFirmwareDB_Count(t *testing.T) {
	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	count := db.Count()
	if count == 0 {
		t.Fatal("expected count > 0")
	}

	if count != len(db.Firmwares) {
		t.Errorf("expected count %d, got %d", len(db.Firmwares), count)
	}
}

func TestFirmwareDB_GetVerified(t *testing.T) {
	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	verified := db.GetVerified()
	if verified == nil {
		t.Fatal("expected non-nil verified slice")
	}

	// All returned firmwares should be verified
	for _, fw := range verified {
		if !fw.Verified {
			t.Errorf("expected firmware %s to be verified", fw.Version)
		}
	}

	// Should be at least one verified firmware (0x355)
	if len(verified) == 0 {
		t.Error("expected at least one verified firmware")
	}
}

func TestFirmware_String(t *testing.T) {
	tests := []struct {
		name     string
		firmware Firmware
		contains []string
	}{
		{
			name: "verified firmware",
			firmware: Firmware{
				Version:  "0x355",
				Name:     "Test Firmware",
				Verified: true,
			},
			contains: []string{"0x355", "Test Firmware", "verified"},
		},
		{
			name: "unverified firmware",
			firmware: Firmware{
				Version:  "0x400",
				Name:     "Unknown Firmware",
				Verified: false,
			},
			contains: []string{"0x400", "Unknown Firmware"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.firmware.String()
			for _, expected := range tt.contains {
				if !strings.Contains(str, expected) {
					t.Errorf("expected String() to contain %q, got: %s", expected, str)
				}
			}
		})
	}
}

func TestFirmware_FormatAddresses(t *testing.T) {
	firmware := Firmware{
		Version: "0x355",
		Functions: FirmwareFunctions{
			SlFsOpen:  0x20015c64,
			SlFsRead:  0x20014b54,
			SlFsWrite: 0x20014bf8,
			SlFsClose: 0x2001555c,
			SlFsDel:   0x20016ea8,
			UartLog:   0x20014f14,
		},
		Memory: FirmwareMemory{
			WorkBuffer:    0x20030000,
			FileHandlePtr: 0x20031000,
			FilenamePtr:   0x20031004,
			TokenPtr:      0x20031020,
			StackBase:     0x20031d00,
		},
	}

	formatted := firmware.FormatAddresses()

	// Check for all expected addresses
	expectedAddresses := []string{
		"0x20015c64", // sl_FsOpen
		"0x20014b54", // sl_FsRead
		"0x20014bf8", // sl_FsWrite
		"0x2001555c", // sl_FsClose
		"0x20016ea8", // sl_FsDel
		"0x20014f14", // uart_log
		"0x20030000", // work_buffer
		"0x20031000", // file_handle_ptr
		"0x20031004", // filename_ptr
		"0x20031020", // token_ptr
		"0x20031d00", // stack_base
	}

	for _, addr := range expectedAddresses {
		if !strings.Contains(formatted, addr) {
			t.Errorf("expected FormatAddresses() to contain %q, got: %s", addr, formatted)
		}
	}

	// Check for section headers
	if !strings.Contains(formatted, "Function Addresses") {
		t.Error("expected FormatAddresses() to contain 'Function Addresses'")
	}
	if !strings.Contains(formatted, "Memory Locations") {
		t.Error("expected FormatAddresses() to contain 'Memory Locations'")
	}
}

func TestHandleUnknownFirmware(t *testing.T) {
	err := HandleUnknownFirmware("0xunknown")
	if err == nil {
		t.Fatal("expected error for unknown firmware")
	}

	// Should be FirmwareUnsupportedError
	if !strings.Contains(err.Error(), "unsupported firmware version") {
		t.Errorf("expected error to contain 'unsupported firmware version', got: %v", err)
	}

	// Error message should contain the version
	if !strings.Contains(err.Error(), "0xunknown") {
		t.Errorf("expected error to contain version '0xunknown', got: %v", err)
	}

	// Error message should contain instructions
	expectedInstructions := []string{
		"dump",
		"submit",
		"issue",
	}

	errMsg := err.Error()
	for _, instruction := range expectedInstructions {
		if !strings.Contains(strings.ToLower(errMsg), instruction) {
			t.Errorf("expected error message to contain %q, got: %v", instruction, err)
		}
	}

	// Try to extract as FirmwareUnsupportedError
	if strings.Contains(err.Error(), "unsupported firmware") {
		// Check that it contains the version we passed
		if !strings.Contains(err.Error(), "0xunknown") {
			t.Error("error message should contain the firmware version")
		}
	}
}

func TestFirmware_StructureComplete(t *testing.T) {
	// Test that loaded firmware has all required fields
	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	fw, ok := db.Get("0x355")
	if !ok {
		t.Skip("firmware 0x355 not found, skipping structure test")
	}

	// Check all fields are populated
	if fw.Version == "" {
		t.Error("expected Version to be set")
	}

	if fw.Name == "" {
		t.Error("expected Name to be set")
	}

	// Check function addresses are non-zero
	if fw.Functions.SlFsOpen == 0 {
		t.Error("expected SlFsOpen to be non-zero")
	}

	if fw.Functions.SlFsWrite == 0 {
		t.Error("expected SlFsWrite to be non-zero")
	}

	if fw.Functions.SlFsClose == 0 {
		t.Error("expected SlFsClose to be non-zero")
	}

	if fw.Functions.SlFsDel == 0 {
		t.Error("expected SlFsDel to be non-zero")
	}

	// Check memory locations are non-zero
	if fw.Memory.WorkBuffer == 0 {
		t.Error("expected WorkBuffer to be non-zero")
	}

	if fw.Memory.FileHandlePtr == 0 {
		t.Error("expected FileHandlePtr to be non-zero")
	}
}

func TestFirmware_0x355_Exists(t *testing.T) {
	// Specific test for the known working firmware version
	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	fw, ok := db.Get("0x355")
	if !ok {
		t.Fatal("expected firmware version 0x355 to exist in catalog")
	}

	if fw.Version != "0x355" {
		t.Errorf("expected version '0x355', got %s", fw.Version)
	}

	// This should be the verified Smartap 0x355 firmware
	if !strings.Contains(fw.Name, "Smartap") {
		t.Errorf("expected name to contain 'Smartap', got %s", fw.Name)
	}

	// Should be verified
	if !fw.Verified {
		t.Error("expected 0x355 to be verified")
	}

	// Check critical function addresses match the working script
	expectedAddresses := map[string]int64{
		"sl_FsOpen":  0x20015c64,
		"sl_FsWrite": 0x20014bf8,
		"sl_FsClose": 0x2001555c,
		"sl_FsDel":   0x20016ea8,
	}

	if fw.Functions.SlFsOpen != expectedAddresses["sl_FsOpen"] {
		t.Errorf("expected sl_FsOpen 0x%x, got 0x%x", expectedAddresses["sl_FsOpen"], fw.Functions.SlFsOpen)
	}

	if fw.Functions.SlFsWrite != expectedAddresses["sl_FsWrite"] {
		t.Errorf("expected sl_FsWrite 0x%x, got 0x%x", expectedAddresses["sl_FsWrite"], fw.Functions.SlFsWrite)
	}

	if fw.Functions.SlFsClose != expectedAddresses["sl_FsClose"] {
		t.Errorf("expected sl_FsClose 0x%x, got 0x%x", expectedAddresses["sl_FsClose"], fw.Functions.SlFsClose)
	}

	if fw.Functions.SlFsDel != expectedAddresses["sl_FsDel"] {
		t.Errorf("expected sl_FsDel 0x%x, got 0x%x", expectedAddresses["sl_FsDel"], fw.Functions.SlFsDel)
	}

	// Check work buffer address
	if fw.Memory.WorkBuffer != 0x20030000 {
		t.Errorf("expected work_buffer 0x20030000, got 0x%x", fw.Memory.WorkBuffer)
	}
}

func TestFirmwareDB_ConcurrentAccess(t *testing.T) {
	// Test that FirmwareDB is safe for concurrent access
	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	done := make(chan bool)

	// Start multiple goroutines accessing the database
	for i := 0; i < 10; i++ {
		go func() {
			// Get firmware
			_, _ = db.Get("0x355")

			// List versions
			_ = db.Versions()

			// Get verified
			_ = db.GetVerified()

			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestFirmware_0x355_Signatures(t *testing.T) {
	// Test that firmware 0x355 has all required signatures for detection
	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	fw, ok := db.Get("0x355")
	if !ok {
		t.Fatal("expected firmware version 0x355 to exist in catalog")
	}

	// Debug: print what we got
	t.Logf("Firmware version: %s", fw.Version)
	t.Logf("Signatures: %v", fw.Functions.Signatures)
	t.Logf("Signature count: %d", len(fw.Functions.Signatures))
	t.Logf("sl_FsOpen address: 0x%x", fw.Functions.SlFsOpen)

	// Check that signatures exist
	if fw.Functions.Signatures == nil {
		t.Fatal("expected Signatures to be set for firmware 0x355")
	}

	// Check that all 7 required signatures are present
	requiredSignatures := []string{
		"sl_FsOpen",
		"sl_FsRead",
		"sl_FsWrite",
		"sl_FsClose",
		"sl_FsDel",
		"sl_FsGetInfo",
		"uart_log",
	}

	for _, sigName := range requiredSignatures {
		sig, exists := fw.Functions.Signatures[sigName]
		if !exists {
			t.Errorf("expected signature for %s to exist", sigName)
			continue
		}

		// Check that signature has two 32-bit values
		if len(sig) != 2 {
			t.Errorf("expected signature for %s to have 2 values, got %d", sigName, len(sig))
			continue
		}

		// Check that signature values are non-zero
		if sig[0] == 0 && sig[1] == 0 {
			t.Errorf("expected signature for %s to be non-zero", sigName)
		}
	}

	// Verify specific known signatures for firmware 0x355
	expectedSignatures := map[string][2]uint32{
		"sl_FsOpen":    {0x4606b570, 0x78004818},
		"sl_FsRead":    {0x43f0e92d, 0x48254680},
		"sl_FsWrite":   {0x43f0e92d, 0x48244680},
		"sl_FsClose":   {0x460db5f0, 0x461c4607},
		"sl_FsDel":     {0x4604b510, 0x78004814},
		"sl_FsGetInfo": {0x481d4603, 0x7800b530},
		"uart_log":     {0x1c04b510, 0xe003d007},
	}

	for sigName, expectedSig := range expectedSignatures {
		actualSig, exists := fw.Functions.Signatures[sigName]
		if !exists {
			t.Errorf("expected signature for %s to exist", sigName)
			continue
		}

		if actualSig[0] != expectedSig[0] || actualSig[1] != expectedSig[1] {
			t.Errorf("signature mismatch for %s: expected [0x%08x, 0x%08x], got [0x%08x, 0x%08x]",
				sigName, expectedSig[0], expectedSig[1], actualSig[0], actualSig[1])
		}
	}
}

func TestFirmware_SlFsGetInfo_Address(t *testing.T) {
	// Test that firmware 0x355 has sl_FsGetInfo address set
	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	fw, ok := db.Get("0x355")
	if !ok {
		t.Fatal("expected firmware version 0x355 to exist in catalog")
	}

	// Check that sl_FsGetInfo address is set
	if fw.Functions.SlFsGetInfo == 0 {
		t.Error("expected SlFsGetInfo address to be non-zero")
	}

	// Verify the known address for 0x355
	expectedAddr := int64(0x2001590c)
	if fw.Functions.SlFsGetInfo != expectedAddr {
		t.Errorf("expected sl_FsGetInfo address 0x%x, got 0x%x",
			expectedAddr, fw.Functions.SlFsGetInfo)
	}
}

func TestFirmware_Signatures_BackwardCompatibility(t *testing.T) {
	// Test that firmwares without signatures can still be loaded
	// (backward compatibility with older YAML format)

	// This test assumes the catalog may have firmwares without signatures
	// For now, we just verify that missing signatures don't cause errors

	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	// Try to iterate all firmwares - none should have nil Signatures field
	// (it should be an empty map if not specified in YAML)
	for _, fw := range db.Firmwares {
		// Signatures can be nil or empty map, both are valid
		if fw.Functions.Signatures != nil && len(fw.Functions.Signatures) > 0 {
			// If signatures exist, validate their format
			for sigName, sig := range fw.Functions.Signatures {
				if len(sig) != 2 {
					t.Errorf("firmware %s: signature %s has invalid length %d (expected 2)",
						fw.Version, sigName, len(sig))
				}
			}
		}
	}
}

func TestFirmware_Signatures_Count(t *testing.T) {
	// Test that all firmwares with signatures have the expected count
	db, err := LoadFirmwares()
	if err != nil {
		t.Fatalf("LoadFirmwares failed: %v", err)
	}

	fw, ok := db.Get("0x355")
	if !ok {
		t.Fatal("expected firmware version 0x355 to exist in catalog")
	}

	// Should have exactly 7 signatures for detection
	expectedCount := 7
	actualCount := len(fw.Functions.Signatures)
	if actualCount != expectedCount {
		t.Errorf("expected %d signatures for firmware 0x355, got %d",
			expectedCount, actualCount)
	}
}
