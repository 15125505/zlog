/**
* 作    者: 雾影
* 创建日期: 2016/9/22
* 功能说明：基于go语言的日志模块
* 当前版本：1.0.0
 */

package log

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	LevelError = iota
	LevelNotice
	LevelInformational
	LevelDebug
)

const maxChanCount = 10000 // 缓冲区最多存放10000条数据

var LevelColor []string // 颜色列表

// 初始化函数
func init() {

	// 缺省生成一个仅仅在控制台输出的日志模块
	Log = NewLogger()
	Log.SetCallLevel(3)

	// 颜色列表
	LevelColor = []string{
		"\033[31m", // 红色
		"\033[33m", // 黄色
		"\033[32m", // 绿色
		"\033[37m", // 白色
	}

}

// ZLogger 日志模块
type ZLogger struct {
	prefix          string // 用户设定的名称
	level           int    // 记入日志文件的等级（小于等于该等级的才计入日志文件）
	isAsyncToFile   bool   // 写入文件的方式（true为异步写入）
	isFileWithColor bool   // 日志文件是否写入颜色信息
	isConsoleOut    bool   // 日志是否输出到控制台
	isHaveErrFile   bool   // 日志是否加入单独的错误日志文件
	isFileDaily     bool   // 日志文件是否按日期命名

	pFile    *os.File // 当前日志文件句柄
	fileName string   // 当前使用的文件名

	pErrFile    *os.File // 当前错误日志文件句柄
	errFileName string   // 当前使用的错误日志文件名

	toWrite   chan LogNode // 需要写入文件的日志
	callLevel int          // 调用级别

	lock sync.RWMutex // 防止多个协程写入屏幕日志的时候，会出错乱
}

// NewLogger 创建一个日志模块
func NewLogger() (l *ZLogger) {
	l = &ZLogger{
		isAsyncToFile:   false,
		isFileWithColor: false,
		isConsoleOut:    true,
		isHaveErrFile:   false,
		isFileDaily:     true,
		callLevel:       2,
		level:           LevelInformational,
		toWrite:         make(chan LogNode, maxChanCount),
		lock:            sync.RWMutex{},
	}
	go l.run()
	return
}

// SetLogFile 日志文件名设置(默认为空)
// 如果日志文件名为空，那么不会输出日志文件
// 配置示例，如果用户如下配置：
//  SetLogFile("logfiles/abc")
// 那么生成的日志文件将如下所示：
// logfiles/abc-20160908.log
// 如果有错误日志生成，将如下所示：
// logfiles/abc-err-20160908.log
// 如果设定的文件所在目录不存在，日志模块会在输出日志的时候自动创建该目录
func (l *ZLogger) SetLogFile(fileName string) {
	l.prefix = fileName
}

// SetLogLevel 设置记录到文件的日志等级（默认只记录info以及以上级别的日志到文件中）
func (l *ZLogger) SetLogLevel(level int) {
	l.level = level
}

// SetConsoleOut 是否允许控制台输出（默认输出到控制台）
func (l *ZLogger) SetConsoleOut(enable bool) {
	l.isConsoleOut = enable
}

// SetFileColor 是否允许文件中带颜色信息（默认文件输出不带颜色）
func (l *ZLogger) SetFileColor(enable bool) {
	l.isFileWithColor = enable
}

// SetWriteFileMode 配置写入日志文件的方式：同步还是异步(默认为同步)
// 同步，意味着实时写入文件
// 异步，则将日志加入缓冲区，由专门的协程写入文件
// 异步的优点：可以瞬间输出更多的日志文件，而且不会阻塞调用者
// 异步的缺点：如果软件crash掉，可能来不及将最后的几条日志写入文件
// 异步的另外一个缺点，是如果输入日志比写入文件的速度快，那么缓冲区会上涨，上涨到满之后会丢日志
func (l *ZLogger) SetWriteFileMode(isAsynToFile bool) {
	l.isAsyncToFile = isAsynToFile
}

// SetAdditionalErrorFile 是否允许错误日志额外存一份文件（默认不单独存错误日志）
// 注意：如果错误日志单独存储，那么每一条错误日志会存两份
func (l *ZLogger) SetAdditionalErrorFile(has bool) {
	l.isHaveErrFile = has
}

// SetCallLevel 设置回调层次(默认为2）
// 设置回调层次的目的是为了能够正确输出调用日志的位置(文件和行号)
// 如果对本日志模块进行了进一步的封装，那么为了正确输出调用日志的位置，需要相应设置回调层次
// 对本模块的封装，每增加一次调用，那么该值需要加1
func (l *ZLogger) SetCallLevel(level int) {
	l.callLevel = level
}

// SetFileDaily 设置是否将日志文件按天存储(默认为true)
func (l *ZLogger) SetFileDaily(yes bool) {
	l.isFileDaily = yes
}

// LogNode 需要写入文件的节点
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
			fmt.Sprintln(node.when.Format("2006-01-02 15:04:05"), LevelColor[(node.level-LevelError)%len(LevelColor)]+node.msg+"\033[0m"),
			"",
			node.when)
	} else {
		err = l.msg2File(
			&l.pFile,
			&l.fileName,
			fmt.Sprintln(node.when.Format("2006-01-02 15:04:05"), node.msg),
			"",
			node.when)
	}

	// 写入日志到错误日志文件
	if l.isHaveErrFile && node.level <= LevelError {
		err = l.msg2File(
			&l.pErrFile,
			&l.errFileName,
			fmt.Sprintln(node.when.Format("2006-01-02 15:04:05"), node.msg),
			"-error",
			node.when)
	}
	return
}

func (l *ZLogger) msg2File(ppFile **os.File, fileName *string, txt, tag string, when time.Time) (err error) {

	// 如果文件名发生变化，需要关闭之前的文件
	var newFileName string
	if l.isFileDaily {
		newFileName = fmt.Sprintf("%v%v-%v.log", l.prefix, tag, when.Format("20060102"))
	} else {
		newFileName = fmt.Sprintf("%v%v.log", l.prefix, tag)
	}
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
		f, err := os.OpenFile(newFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
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

// Log 缺省的日志输出
var Log *ZLogger

func (l *ZLogger) msgOut(logLevel int, txt string) {
	now := time.Now()

	// 找出调用所在的文件和代码行
	_, file, line, ok := runtime.Caller(l.callLevel)
	if !ok {
		file = "N/A"
		line = 0
	}
	_, filename := path.Split(file)
	txt = " " + filename + ":" + strconv.FormatInt(int64(line), 10) + " " + txt

	// 输出到控制台
	if l.isConsoleOut {

		l.lock.Lock()

		// 输出日期
		fmt.Print(now.Format("2006-01-02 15:04:05"))

		// 设置颜色
		ColorBegin(logLevel)

		// 输出内容
		fmt.Print(txt)

		// 结束颜色设置
		ColorEnd()

		// 输出换行
		fmt.Print("\n")

		l.lock.Unlock()
	}

	// 大于指定等级或者日志文件名为空，不输出到文件
	if logLevel > l.level || l.prefix == "" {
		return
	}
	if l.isAsyncToFile {
		if len(l.toWrite) < maxChanCount {
			l.toWrite <- LogNode{when: now, msg: txt, level: logLevel}
		}
	} else {
		l.msgToFile(LogNode{when: now, msg: txt, level: logLevel})
	}
}

// 错误级别日志
func (l *ZLogger) Error(v ...interface{}) {
	msg := fmt.Sprintf("[E] "+l.formatMsg(len(v)), v...)
	l.msgOut(LevelError, msg)
}

// 提醒级别日志
func (l *ZLogger) Notice(v ...interface{}) {
	msg := fmt.Sprintf("[N] "+l.formatMsg(len(v)), v...)
	l.msgOut(LevelNotice, msg)
}

// 信息级别日志
func (l *ZLogger) Informational(v ...interface{}) {
	msg := fmt.Sprintf("[I] "+l.formatMsg(len(v)), v...)
	l.msgOut(LevelInformational, msg)
}

// 信息级别日志
func (l *ZLogger) Info(v ...interface{}) {
	msg := fmt.Sprintf("[I] "+l.formatMsg(len(v)), v...)
	l.msgOut(LevelInformational, msg)
}

// 调试级别日志
func (l *ZLogger) Debug(v ...interface{}) {
	msg := fmt.Sprintf("[D] "+l.formatMsg(len(v)), v...)
	l.msgOut(LevelDebug, msg)
}

// 调试级别日志
func Debug(v ...interface{}) {
	Log.Debug(v...)
}

// 信息级别日志
func Info(v ...interface{}) {
	Log.Info(v...)
}

// 提醒级别日志
func Notice(v ...interface{}) {
	Log.Notice(v...)
}

// 错误级别日志
func Error(v ...interface{}) {
	Log.Error(v...)
}

// 生成msg字符串
func (l *ZLogger) formatMsg(n int) string {
	return strings.Repeat("%v ", n)
}
