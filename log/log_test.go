package log

import (
	"testing"
	"time"
	"fmt"
)


func TestMsgToFile(t *testing.T) {
	l := NewLogger()
	l.SetLogFile("tmp/msgToFile")
	l.SetAdditionalErrorFile(true)
	l.msgToFile(LogNode{ time.Now(), "控制台和文件中的 debug ", LevelDebug})
	l.msgToFile(LogNode{ time.Now(), "控制台和文件中的 info", LevelInformational})
	l.msgToFile(LogNode{ time.Now(), "控制台和文件中的 notice", LevelNotice})
	l.msgToFile(LogNode{ time.Now(), "控制台和文件中的 error（错误文件中也应该有）", LevelError})

	fmt.Println("检查一下tmp目录下对应的两个文件")
}

func TestMsgOut(t *testing.T) {
	l := NewLogger()
	l.SetConsoleOut(false)
	l.SetLogFile("tmp/msgOut")

	l.msgOut(LevelDebug, "不应该出现在控制台的 debug")
	l.msgOut(LevelInformational, "不应该出现在控制台的 info")
	l.msgOut(LevelNotice, "不应该出现在控制台的 notice")
	l.msgOut(LevelError, "不应该出现在控制台的 error")


	fmt.Println("检查一下tmp目录下对应的msgOut文件")
}

func TestOut(t *testing.T) {
	Log.SetLogFile("tmp/color")
	Log.SetFileColor(true)
	Log.SetFileDaily(false)

	Debug("文件中没有的 debug")
	Info("文件中有颜色的 info")
	Notice("文件中有颜色的 notice")
	Error("文件中有颜色的 error")

	Log.SetFileColor(false)
	Log.SetLogLevel(LevelDebug)

	Debug("文件中没有颜色的 debug")
	Info("文件中没有颜色的 info")
	Notice("文件中没有颜色的 notice")
	Error("文件中没有颜色的 error")
	time.Sleep(time.Second)

	fmt.Println("检查一下tmp目录下对应的color文件")

}