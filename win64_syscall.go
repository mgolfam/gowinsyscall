package gowinsyscall

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const (
	SW_RESTORE = 9
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	enumWindowsProc     = user32.NewProc("EnumWindows")
	getWindowThreadProc = user32.NewProc("GetWindowThreadProcessId")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
	showWindow          = user32.NewProc("ShowWindow")
	getWindowTextW      = user32.NewProc("GetWindowTextW")
)

type WindowInfo struct {
	Title string
	PID   uint32
}

type EnumFunc func(hwnd syscall.Handle, lParam uintptr) uintptr

func EnumWindows(enumFunc EnumFunc, lParam uintptr) error {
	ret, _, err := enumWindowsProc.Call(syscall.NewCallback(enumFunc), lParam)
	if ret == 0 {
		if err.Error() != "The operation completed successfully." {
			return err
		}
	}
	return nil
}

func SetForegroundWindowByPID(pid uint32) error {
	var hWnd syscall.Handle
	enumFunc := func(hwnd syscall.Handle, lParam uintptr) uintptr {
		var processID uint32
		_, _, _ = getWindowThreadProc.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&processID)))
		if processID == pid {
			hWnd = hwnd
			return 0 // Stop enumeration
		}
		return 1 // Continue enumeration
	}

	err := EnumWindows(EnumFunc(enumFunc), 0)
	if err != nil {
		return err
	}
	if hWnd == 0 {
		return fmt.Errorf("Window with PID %d not found", pid)
	}
	ret, _, err := setForegroundWindow.Call(uintptr(hWnd))
	if ret == 0 {
		return err
	}
	ret, _, err = showWindow.Call(uintptr(hWnd), SW_RESTORE)
	if ret == 0 {
		return err
	}
	return nil
}

func EnumrateWindows(callback func(WindowInfo) bool) error {
	cb := syscall.NewCallback(func(hwnd syscall.Handle, lparam uintptr) uintptr {
		title, err := GetWindowText(hwnd)
		if err != nil {
			// Ignore the error
			return 1 // continue enumeration
		}
		processID, _ := GetWindowProcessID(hwnd)
		info := WindowInfo{
			Title: title,
			PID:   processID,
		}
		if !callback(info) {
			return 0 // stop enumeration
		}
		return 1 // continue enumeration
	})

	r1, _, e1 := syscall.Syscall(enumWindowsProc.Addr(), 2, uintptr(cb), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			return error(e1)
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetWindowText(hwnd syscall.Handle) (string, error) {
	const nChars = 1024
	var buf [nChars]uint16
	_, _, err := syscall.Syscall(getWindowTextW.Addr(), 3,
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)))
	if err != 0 {
		return "", err
	}
	return syscall.UTF16ToString(buf[:]), nil
}

func GetWindowProcessID(hwnd syscall.Handle) (uint32, error) {
	var processID uint32
	r1, _, err := getWindowThreadProc.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&processID)))
	if r1 == 0 {
		return 0, err
	}
	return processID, nil
}

func ListAllWindows() {
	err := EnumrateWindows(func(info WindowInfo) bool {
		fmt.Printf("Window: %s, PID: %d\n", info.Title, info.PID)
		return true // continue enumeration
	})
	if err != nil {
		fmt.Println("Error enumerating windows:", err)
	}
}

func FindWindowsPidByTitle(titleName string) {
	err := EnumrateWindows(func(info WindowInfo) bool {
		// fmt.Printf("Window: %s, PID: %d\n", info.Title, info.PID)
		return true // continue enumeration
	})
	if err != nil {
		// log.Fatal("Error enumerating windows:", err)
		fmt.Println(err)
	}
}

func SelectPidByTitle(titleName string) {
	err := EnumrateWindows(func(info WindowInfo) bool {
		// fmt.Printf("Window: %s, PID: %d\n", info.Title, info.PID)

		if strings.Contains(strings.ToLower(info.Title), strings.ToLower(titleName)) {
			pid := info.PID
			if err := SetForegroundWindowByPID(pid); err != nil {
				// log.Fatal(err)
				fmt.Println(err)
			}
			// fmt.Println("Successfully set focus to a window associated with PID", pid)
		}

		return true // continue enumeration
	})
	if err != nil {
		fmt.Println("Error enumerating windows:", err)
	}
}
