// +build windows

package database

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// checkDiskSpace checks if there's sufficient disk space available
func checkDiskSpace(path string, requiredBytes int64) error {
	return checkDiskSpaceWindows(path, requiredBytes)
}

// checkDiskSpaceWindows checks disk space on Windows using GetDiskFreeSpaceEx
func checkDiskSpaceWindows(path string, requiredBytes int64) error {
	// Convert path to absolute path if needed
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get the root directory (e.g., C:\ for Windows)
	rootPath := filepath.VolumeName(absPath)
	if rootPath == "" {
		return fmt.Errorf("failed to determine volume for path: %s", absPath)
	}
	rootPath += string(filepath.Separator)

	// Windows API call to get free disk space
	// GetDiskFreeSpaceEx retrieves disk space information
	kernel32, err := windows.LoadLibrary("Kernel32.dll")
	if err != nil {
		return fmt.Errorf("failed to load kernel32: %w", err)
	}
	defer windows.FreeLibrary(kernel32)

	getDiskFreeSpaceEx, err := windows.GetProcAddress(kernel32, "GetDiskFreeSpaceExW")
	if err != nil {
		return fmt.Errorf("failed to get GetDiskFreeSpaceExW address: %w", err)
	}

	var freeBytesAvailable int64
	var totalNumberOfBytes int64
	var totalNumberOfFreeBytes int64

	// Call GetDiskFreeSpaceExW
	// We need to convert the path to a UTF-16 pointer
	pathPtr, err := windows.UTF16PtrFromString(rootPath)
	if err != nil {
		return fmt.Errorf("failed to convert path to UTF-16: %w", err)
	}

	// Using syscall to call the Windows API
	syscall.Syscall6(uintptr(getDiskFreeSpaceEx),
		4,
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
		0,
		0)

	if freeBytesAvailable < requiredBytes {
		return fmt.Errorf("insufficient disk space: %d bytes required, %d available", requiredBytes, freeBytesAvailable)
	}

	return nil
}
