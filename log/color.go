// +build !windows

package log

import "fmt"

func ColorBegin(logLevel int) {
	fmt.Print(LevelColor[(logLevel-LevelError)%len(LevelColor)])
}

func ColorEnd()  {
	fmt.Print("\033[0m")
}