//go:build windows

package health

import "golang.org/x/sys/windows"

func freeBytes(path string) (uint64, error) {
	var freeAvail, totalBytes, totalFree uint64
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	if err := windows.GetDiskFreeSpaceEx(ptr, &freeAvail, &totalBytes, &totalFree); err != nil {
		return 0, err
	}
	return freeAvail, nil
}
