package log

import (
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	NewLogger(true, true, false, true, "abc/fff")
}

func TestMsgToFile(t *testing.T) {
	l := NewLogger(true, true, false, true, "tmp/msgToFile")
	l.msgToFile(LogNode{ time.Now(), "test err message ", LevelError})
	l.msgToFile(LogNode{ time.Now(), "test info message ", LevelInformational})
	l.msgToFile(LogNode{ time.Now(), "test debug message ", LevelDebug})
	t.Log("检查一下tmp目录下是否有对应的两个文件")
}

func TestMsgOut(t *testing.T) {
	l := NewLogger(true, true, false, true, "tmp/msgOut")
	l.msgOut(LevelDebug, "msg debug msg")
	l.msgOut(LevelInformational, "msg info msg")
	l.msgOut(LevelError, "msg error msg")
	t.Log("检查一下tmp目录下是否有对应的msgOut文件")
}

func TestOut(t *testing.T) {
	Log.Config(true, false, true, true, "tmp/noColor")
	Debug("debug info")
	Info("info")
	Error("error")
	Log.Config(true, true, false, false, "tmp/color")
	Debug("debug info")
	Info("info")
	Error("error")
	time.Sleep(time.Second)
}