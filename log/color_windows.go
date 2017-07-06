package log

import (
	"fmt"
	"syscall"
	"unsafe"
)

var WinLevelColor []uint16 // windows下的颜色列表

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	SetConsoleTextAttribute    = kernel32.NewProc("SetConsoleTextAttribute")
	GetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	isWinConsole               = false
)

const (
	blue      = uint16(1)
	green     = uint16(2)
	red       = uint16(4)
	intensity = uint16(8)
)

func init() {
	WinLevelColor = []uint16{
		red | intensity,
		red | green,
		green,
		red | green | blue,
	}
	isWinConsole = isInwinConsole(uintptr(syscall.Stdout))
}
func ColorBegin(logLevel int) {
	if !isWinConsole {
		fmt.Print(LevelColor[(logLevel-LevelError)%len(LevelColor)])
	} else {
		SetConsoleTextAttribute.Call(uintptr(syscall.Stdout), uintptr(WinLevelColor[(logLevel-LevelError)%len(WinLevelColor)]))
	}
}

func ColorEnd() {
	if !isWinConsole {
		fmt.Print("\033[0m")
	} else {
		SetConsoleTextAttribute.Call(uintptr(syscall.Stdout), uintptr(red|green|blue))
	}
}

// 获取当前是否在windows控制台
func isInwinConsole(hConsoleOutput uintptr) (isInWinConsole bool) {
	if nil == GetConsoleScreenBufferInfo {
		return false
	}
	csbi := struct {
		DwSize              int32
		DwCursorPosition    int32
		WAttributes         uint16
		SrWindow            [4]int16
		DwMaximumWindowSize int32
	}{}
	ret, _, _ := GetConsoleScreenBufferInfo.Call(
		hConsoleOutput,
		uintptr(unsafe.Pointer(&csbi)))
	return ret != 0
}
