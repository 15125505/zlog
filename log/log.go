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

const maxChanCount = 10000   // 缓冲区最多存放10000条数据
var LevelColor []string     // 颜色列表

// 初始化函数
func init() {
	// 缺省生成一个仅仅在控制台输出的日志模块
	Log = NewLogger()
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

// 创建一个日志模块
func NewLogger() (l *ZLogger) {
	l = &ZLogger{
		isAsynToFile:false,
		isFileWithColor:false,
		isConsoleOut:true,
		isHaveErrFile:false,
		callLevel:2,
		level:LevelInformational,
		toWrite:make(chan LogNode, maxChanCount),
	}
	go l.run()
	return
}


// 是否允许控制台输出（默认输出到控制台）
func (l *ZLogger)SetConsoleOut(enable bool)  {
	l.isConsoleOut = enable
}

// 是否允许文件中带颜色信息（默认文件输出不带颜色）
func (l *ZLogger)SetFileColor(enable bool)  {
	l.isFileWithColor = enable
}

// 配置写入日志文件的方式：同步还是异步(默认为同步)
// 同步，意味着实时写入文件
// 异步，则将日志加入缓冲区，由专门的协程写入文件
// 异步的优点：可以瞬间输出更多的日志文件，而且不会阻塞调用者
// 异步的缺点：如果软件crash掉，可能来不及将最后的几条日志写入文件
// 异步的另外一个缺点，是如果输入日志比写入文件的速度快，那么缓冲区会上涨，上涨到满之后会丢日志
func (l *ZLogger)SetWriteFileMode(isAsynToFile bool)  {
	l.isAsynToFile = isAsynToFile
}

// 是否允许错误日志额外存一份文件（默认不单独存错误日志）
// 注意：如果错误日志单独存储，那么每一条错误日志会存两份
func (l *ZLogger)SetAdditionalErrorFile(has bool)  {
	l.isHaveErrFile = has
}

// 日志文件名设置(默认为空)
// 如果日志文件名为空，那么不会输出日志文件
// 配置示例，如果用户如下配置：
//  SetLogFile("logfiles/abc")
// 那么生成的日志文件将如下所示：
// logfiles/abc-2016-09-08.log
// 如果有错误日志生成，将如下所示：
// logfiles/abc-err-2016-09-08.log
// 如果设定的文件所在目录不存在，日志模块会在输出日志的时候自动创建该目录
func (l *ZLogger)SetLogFile(fileName string)  {
	l.prefix = fileName
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
		*ppFile = nil
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