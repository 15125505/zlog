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
	LevelError = iota
	LevelInformational
	LevelDebug
)

const maxChanCount = 1000   // 缓冲区最多存放1000条数据
var LevelColor []string     // 颜色列表

// 初始化函数
func init() {
	// 缺省生成一个仅仅在控制台输出的日志模块
	Log = NewLogger(true, false, false, true, "")
	Log.callLevel ++;

	// 颜色列表
	LevelColor = []string{
		"\033[31m",
		"\033[32m",
		"\033[37m",
	}
}

// 日志模块
type ZLogger struct {
	prefix          string       // 用户设定的名称
	level           int          // 记入日志文件的等级（小于等于该等级的才计入日志文件）
	isAsynToFile    bool         // 写入文件的方式（true为异步写入）
	isFileWithColor bool         // 日志文件是否写入颜色信息
	isConsoleOut    bool         // 日志是否输出到控制台
	isHaveErrFile   bool         // 日志是否加入单独的错误日志文件

	pFile           *os.File     // 当前日志文件句柄
	fileName        string       // 当前使用的文件名

	pErrFile        *os.File     // 当前错误日志文件句柄
	errFileName     string       // 当前使用的错误日志文件名

	toWrite         chan LogNode // 需要写入文件的日志
	callLevel       int          // 调用级别
}

// 创建一个日志模块（name示例："logs/myModuleName"）
// console          -- 是否允许控制台输出
// fileWithColor    -- 输出到文件时是否需要带上控制台颜色标志（只有在允许写文件的时候有效）
// asynWrite        -- ture表示异步写入文件，为false表示同步写入文件（只有在允许写文件的时候有效）
// errFile          -- ture表示为错误日志额外写入一份文件（错误日志同时也会写入普通文件），文件名如"logs/mylog-err-20160906.log"
// fileName         -- 日志文件名，如文件名设置为"logs/mylog"，生成的日志文件将如"logs/mylog-20160906.log"，
//                     如果fileName为空，表示日志不写入文件
func NewLogger(console, fileWithColor, asynWrite, errFile bool, fileName string) (l *ZLogger) {
	l = &ZLogger{
		callLevel:2,
		level:LevelInformational,
		toWrite:make(chan LogNode, maxChanCount),
	}
	l.Config(console, fileWithColor, asynWrite, errFile, fileName)
	go l.run()
	return
}

// 修改配置信息
func (l *ZLogger)Config(console, fileWithColor, asynWrite, errFile bool, fileName string) {
	l.prefix = fileName
	l.isFileWithColor = fileWithColor
	l.isAsynToFile = asynWrite
	l.isHaveErrFile = errFile
	l.isConsoleOut = console
}

// 需要写入文件的节点
type LogNode struct {
	when  time.Time
	msg   string
	level int
}

// 用于写文件的协程
func (l *ZLogger) run() {
	for {
		select {
		case node := <-l.toWrite:
			l.msgToFile(node)
			n := len(l.toWrite)
			for i := 0; i < n; i++ {
				l.msgToFile(<-l.toWrite)
			}
		}
	}
}

// 实际的写文件函数
func (l *ZLogger) msgToFile(node LogNode) (err error) {

	// 写入日志到文件
	if l.isFileWithColor {
		err = l.msg2File(
			&l.pFile,
			&l.fileName,
			fmt.Sprintln(node.when.Format("15:04:05"), LevelColor[(node.level - LevelError) % len(LevelColor)], node.msg, "\033[0m"),
			"",
			node.when)
	} else {
		err = l.msg2File(
			&l.pFile,
			&l.fileName,
			fmt.Sprintln(node.when.Format("15:04:05"), node.msg),
			"",
			node.when)
	}

	// 写入日志到错误日志文件
	if l.isHaveErrFile && node.level <= LevelError {
		err = l.msg2File(
			&l.pErrFile,
			&l.errFileName,
			fmt.Sprintln(node.when.Format("15:04:05"), node.msg),
			"-err",
			node.when)
	}

	return
}

func (l *ZLogger)msg2File(ppFile **os.File, fileName *string, txt, tag string, when time.Time) (err error) {

	// 如果文件名发生变化，需要关闭之前的文件
	newFileName := fmt.Sprintf("%v%v-%v.log", l.prefix, tag, when.Format("20060102"))
	if newFileName != *fileName && *ppFile != nil {
		(*ppFile).Close()
		ppFile = nil
	}

	// 如果文件没有打开，首先需要打开文件
	if nil == *ppFile {
		// 创建文件目录
		err := os.MkdirAll(filepath.Dir(newFileName), 0775)
		if err != nil {
			fmt.Println(err)
		}

		// 打开文件
		f, err := os.OpenFile(newFileName, os.O_CREATE | os.O_APPEND | os.O_RDWR, 0664)
		if err != nil {
			fmt.Println("创建日志文件失败！", newFileName)
			return err
		}
		*fileName = newFileName
		*ppFile = f
	}

	// 写入日志到文件
	_, err = (*ppFile).WriteString(txt)
	if err != nil {
		fmt.Println("写入日志文件失败！", newFileName)
		(*ppFile).Close()
		*ppFile = nil
	}
	return
}

// 缺省的日志输出
var Log  *ZLogger

func (l *ZLogger) msgOut(logLevel int, txt string) {
	now := time.Now()

	// 找出调用所在的文件和代码行
	_, file, line, ok := runtime.Caller(l.callLevel)
	if !ok {
		file = "???"
		line = 0
	}
	_, filename := path.Split(file)
	txt = "[" + filename + ":" + strconv.FormatInt(int64(line), 10) + "]" + txt

	// 输出到控制台
	if l.isConsoleOut {
		fmt.Println(now.Format("01-02 15:04:05"), LevelColor[(logLevel - LevelError) % len(LevelColor)], txt, "\033[0m")
	}

	// 大于指定等级或者日志文件名为空，不输出到文件
	if logLevel > l.level || l.prefix == "" {
		return
	}
	if l.isAsynToFile {
		if len(l.toWrite) < maxChanCount {
			l.toWrite <- LogNode{when:now, msg:txt, level:logLevel}
		}
	} else {
		l.msgToFile(LogNode{when:now, msg:txt, level:logLevel})
	}
}


// 错误级别日志
func (l *ZLogger)Error(v ...interface{}) {
	msg := fmt.Sprintf("[E] " + l.generateFmtStr(len(v)), v...)
	l.msgOut(LevelError, msg)
}

// 信息级别日志
func (l *ZLogger)Informational(v ...interface{}) {
	msg := fmt.Sprintf("[I] " + l.generateFmtStr(len(v)), v...)
	l.msgOut(LevelInformational, msg)
}

// 信息级别日志
func (l *ZLogger)Info(v ...interface{}) {
	msg := fmt.Sprintf("[I] " + l.generateFmtStr(len(v)), v...)
	l.msgOut(LevelInformational, msg)
}

// 调试级别日志
func (l *ZLogger)Debug(v ...interface{}) {
	msg := fmt.Sprintf("[D] " + l.generateFmtStr(len(v)), v...)
	l.msgOut(LevelDebug, msg)
}

// 调试级别日志
func Debug(v ...interface{}) {
	Log.Debug(v ...)
}

// 信息级别日志
func Info(v ...interface{}) {
	Log.Info(v ...)
}

// 错误级别日志
func Error(v ...interface{}) {
	Log.Error(v ...)
}

func (l *ZLogger)generateFmtStr(n int) string {
	return strings.Repeat("%v ", n)
}