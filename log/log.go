package log

import (
	"fmt"
	"strings"
	"os"
	"time"
	"runtime"
	"path"
	"strconv"
	"path/filepath"
)

const (
	LevelEmergency = iota
	LevelAlert
	LevelCritical
	LevelError
	LevelWarning
	LevelNotice
	LevelInformational
	LevelDebug
)

// 初始化函数
func init() {
	// 缺省在logs目录下生成run*.log文件，默认同步写入文件
	Log = NewLogger("logs/run", false)
}


// 日志模块
type ZLogger struct {
	filename    string       // 用户设定的名称
	f           *os.File     // 当前日志文件句柄
	curFileName string       // 当前使用的文件明白
	level       int          // 记入日志文件的等级
	asyn        bool         // 写入文件的方式（true为异步写入）
	toWrite     chan LogNode // 需要写入文件的日志
}

// 创建一个日志模块（name示例："logs/myModuleName"）
// fileName     -- 日志文件名，如值为"logs/mylog"，那么将会"logs"目录下生成如"mylog-20160906.log"这样的系列文件
// asynWrite    -- 是否异步写入日志文件，如果为ture表示异步写入，为false表示同步写入文件
func NewLogger(fileName string, asynWrite bool) (l *ZLogger) {
	l = &ZLogger{
		filename : fileName,
		level:LevelDebug,
		asyn:asynWrite,
		toWrite:make(chan LogNode, 1000),
	}
	go l.run()
	return
}

// 修改配置信息
func (l *ZLogger)Config(fileName string, asynWrite bool) {
	l.filename = fileName
	l.asyn = asynWrite
}

// 需要写入文件的节点
type LogNode struct {
	when time.Time
	msg  string
}

// 用于写文件的协程
func (l *ZLogger) run() {
	for {
		select {
		case node := <-l.toWrite:
			l.writeToFile(node)
			n := len(l.toWrite)
			for i := 0; i < n; i++ {
				l.writeToFile(<-l.toWrite)
			}
		}
	}
}

// 实际的写文件函数
func (l *ZLogger) writeToFile(node LogNode) (err error) {

	// 如果文件名发生变化，需要关闭之前的文件
	newFileName := fmt.Sprintf("%v-%v.log", l.filename, node.when.Format("20060102"))
	if newFileName != l.curFileName && l.f != nil {
		l.f.Close()
		l.f = nil
	}

	// 如果文件没有打开，首先需要打开文件
	if nil == l.f {
		// 创建文件目录
		err := os.MkdirAll(filepath.Dir(newFileName), 0600)
		if err != nil {
			fmt.Println(err)
		}

		// 打开文件
		f, err := os.OpenFile(newFileName, os.O_CREATE | os.O_APPEND | os.O_RDWR, 0660)
		if err != nil {
			fmt.Println("创建日志文件失败！", newFileName)
			return err
		}
		l.curFileName = newFileName
		l.f = f
	}

	// 写入日志到文件
	_, err = l.f.WriteString(node.msg + "\n")
	if err != nil {
		fmt.Println("写入日志文件失败！", newFileName)
		l.f.Close()
		l.f = nil
		return err
	}
	return
}

// 缺省的日志输出
var Log  *ZLogger


func (l *ZLogger) writeMsg(logLevel int, txt string) error {
	now := time.Now()

	// 找出调用所在的文件和代码行
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}
	_, filename := path.Split(file)
	txt = "[" + filename + ":" + strconv.FormatInt(int64(line), 10) + "]" + txt

	// 颜色列表
	LevelColor := []string{
		"\033[33m\033[41m",
		"\033[30m\033[45m",
		"\033[30m\033[43m",
		"\033[31m\033[40m",
		"\033[35m\033[40m",
		"\033[36m\033[40m",
		"\033[32m\033[40m",
		"\033[37m\033[40m",
	}

	// 输出到控制台
	txt = fmt.Sprint("\033[30m\033[47m", now.Format("01-02 15:04:05"), LevelColor[(logLevel - LevelEmergency) % len(LevelColor)], txt, "\033[0m")
	fmt.Println(txt)

	// 大于指定等级，不输出到文件
	if logLevel > l.level {
		return nil
	}
	if l.asyn {
		l.toWrite <- LogNode{when : now, msg : txt}
	} else {
		l.writeToFile(LogNode{when : now, msg : txt})
	}
	return nil
}

// Emergency logs a message at emergency level.
func (l *ZLogger)Emergency(v ...interface{}) {
	msg := fmt.Sprintf("[M] " + l.generateFmtStr(len(v)), v...)
	l.writeMsg(LevelEmergency, msg)
}

// Alert logs a message at alert level.
func (l *ZLogger)Alert(v ...interface{}) {
	msg := fmt.Sprintf("[A] " + l.generateFmtStr(len(v)), v...)
	l.writeMsg(LevelAlert, msg)
}

// Critical logs a message at critical level.
func (l *ZLogger)Critical(v ...interface{}) {
	msg := fmt.Sprintf("[C] " + l.generateFmtStr(len(v)), v...)
	l.writeMsg(LevelCritical, msg)
}

// Error logs a message at error level.
func (l *ZLogger)Error(v ...interface{}) {
	msg := fmt.Sprintf("[E] " + l.generateFmtStr(len(v)), v...)
	l.writeMsg(LevelError, msg)
}

// Warning logs a message at warning level.
func (l *ZLogger)Warning(v ...interface{}) {
	msg := fmt.Sprintf("[W] " + l.generateFmtStr(len(v)), v...)
	l.writeMsg(LevelWarning, msg)
}

// Warn compatibility alias for Warning()
func (l *ZLogger)Warn(v ...interface{}) {
	msg := fmt.Sprintf("[W] " + l.generateFmtStr(len(v)), v...)
	l.writeMsg(LevelWarning, msg)
}

// Notice logs a message at notice level.
func (l *ZLogger)Notice(v ...interface{}) {
	msg := fmt.Sprintf("[N] " + l.generateFmtStr(len(v)), v...)
	l.writeMsg(LevelNotice, msg)
}

// Informational logs a message at info level.
func (l *ZLogger)Informational(v ...interface{}) {
	msg := fmt.Sprintf("[I] " + l.generateFmtStr(len(v)), v...)
	l.writeMsg(LevelInformational, msg)
}

// Info compatibility alias for Warning()
func (l *ZLogger)Info(v ...interface{}) {
	msg := fmt.Sprintf("[I] " + l.generateFmtStr(len(v)), v...)
	l.writeMsg(LevelInformational, msg)
}

// Debug logs a message at debug level.
func (l *ZLogger)Debug(v ...interface{}) {
	msg := fmt.Sprintf("[D] " + l.generateFmtStr(len(v)), v...)
	l.writeMsg(LevelDebug, msg)
}

func Debug(v ...interface{}) {
	Log.Debug(v ...)
}

func Info(v ...interface{}) {
	Log.Info(v ...)
}

func Error(v ...interface{}) {
	Log.Error(v ...)
}

func (l *ZLogger)generateFmtStr(n int) string {
	return strings.Repeat("%v ", n)
}