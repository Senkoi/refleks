//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

var trainingUser32 = syscall.NewLazyDLL("user32.dll")
var trainingCreateWindow = trainingUser32.NewProc("CreateWindowExW")
var trainingShowWindow = trainingUser32.NewProc("ShowWindow")
var trainingUpdateWindow = trainingUser32.NewProc("UpdateWindow")
var trainingDestroyWindow = trainingUser32.NewProc("DestroyWindow")
var trainingGetMessage = trainingUser32.NewProc("GetMessageW")
var trainingTranslateMessage = trainingUser32.NewProc("TranslateMessage")
var trainingDispatchMessage = trainingUser32.NewProc("DispatchMessageW")
var trainingSetTimer = trainingUser32.NewProc("SetTimer")
var trainingGetSystemMetrics = trainingUser32.NewProc("GetSystemMetrics")
var trainingMessageBeep = trainingUser32.NewProc("MessageBeep")

type trainingPoint struct{ X, Y int32 }
type trainingMSG struct {
	Window uintptr
	Message uint32
	_ uint32
	WParam uintptr
	LParam uintptr
	Time uint32
	Point trainingPoint
	Private uint32
}

// A non-activating topmost window stays over borderless games without stealing
// mouse/keyboard focus. Exclusive fullscreen can bypass desktop windows.
func showTrainingReminder(message string) {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		trainingMessageBeep.Call(0x30) // Windows warning sound, including exclusive fullscreen.
		class, _ := syscall.UTF16PtrFromString("STATIC")
		label, _ := syscall.UTF16PtrFromString("Refleks 训练提醒\r\n" + message)
		width, _, _ := trainingGetSystemMetrics.Call(0)
		// WS_EX_TOPMOST | WS_EX_TRANSPARENT | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE;
		// WS_POPUP | WS_BORDER | SS_CENTER | SS_CENTERIMAGE.
		window, _, _ := trainingCreateWindow.Call(
			0x00000008|0x00000020|0x00000080|0x08000000,
			uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(label)),
			0x80000000|0x00800000|0x00000001|0x00000200,
			(width-560)/2, 36, 560, 112, 0, 0, 0, 0,
		)
		if window == 0 {
			return
		}
		trainingShowWindow.Call(window, 4) // SW_SHOWNOACTIVATE
		trainingUpdateWindow.Call(window)
		trainingSetTimer.Call(window, 1, 7000, 0)
		var msg trainingMSG
		for {
			result, _, _ := trainingGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if result == 0 || result == ^uintptr(0) || (msg.Window == window && msg.Message == 0x113) {
				break
			}
			trainingTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			trainingDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
		trainingDestroyWindow.Call(window)
	}()
}
