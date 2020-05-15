package xlog

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type XFile struct {
	filename string
	file     *os.File
	*XLogBase
	logChan chan *LogData
	wg *sync.WaitGroup
	curDay int   //也可以按小时去切割，curHour int
}

func (c *XFile) Init() (err error) {
	//初始化日志文件
	c.file,err = os.OpenFile(c.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY,0755)
	if err != nil {
		return
	}
	return
}

func NewXFile(level int, filename, module string) XLog {
	logger := &XFile{
		filename: filename,
	}
	logger.XLogBase = &XLogBase{
		module: module,
		level: level,
	}

	logger.curDay = time.Now().Day()           // 初始化结构体logger.curHour = time.Now().Hour()
	logger.wg = &sync.WaitGroup{}              //防止主进程退出，不执行子进程
	logger.logChan = make(chan *LogData,10000) //管道初始化

	//异步刷日志到磁盘
	logger.wg.Add(1)
	go logger.syncLog()
	return logger
}

func (c *XFile) syncLog() {
	//从管道读取日志，然后写入文件
	for data := range c.logChan {
		c.splitLog()  //调用切分日志的操作
		c.writeLog(c.file,data)
	}
	c.wg.Done()
}

func (c *XFile) splitLog() {
	now := time.Now()
	if now.Day() == c.curDay {
		return
	}
	c.curDay = now.Day() //更新时间 //按小时切分的配置，c.curHour = now.Hour()
	c.file.Sync()
	c.file.Close()

	newFilename := fmt.Sprintf("%s-%04d-%02d-%02d",c.filename,
		now.Year(),now.Month(),now.Day())
	/*
	按小时切分配置
	newFilename := fmt.Sprintf("%s-%04d-%02d-%02d-%02d", c.filename,now.Year(), now.Month(), now.Day(), now.Hour())
	 */
	os.Rename(c.filename,newFilename)
	c.Init()
}

func (c *XFile) writeToChan(level int,module string,format string,args ...interface{})  {
	logData := c.formatLogger(level, module, format, args...)
	select {
	case c.logChan <- logData:
	default:
	}
}

func (c *XFile) LogDebug(format string, args ...interface{}) {
	if c.level > XLogLevelDebug {
		return
	}
	c.writeToChan(XLogLevelDebug, c.module, format, args...)
}

func (c *XFile) LogTrace(format string, args ...interface{}) {
	if c.level > XLogLevelTrace {
		return
	}
	c.writeToChan(XLogLevelTrace, c.module, format, args...)
}

func (c *XFile) LogInfo(format string, args ...interface{}) {
	if c.level > XLogLevelInfo {
		return
	}
	c.writeToChan(XLogLevelInfo, c.module, format, args...)
}

func (c *XFile) LogWarn(format string, args ...interface{}) {
	if c.level > XLogLevelWarn {
		return
	}
	c.writeToChan(XLogLevelWarn, c.module, format, args...)
}

func (c *XFile) LogError(format string, args ...interface{}) {
	if c.level > XLogLevelError {
		return
	}
	c.writeToChan(XLogLevelError, c.module, format, args...)
}

func (c *XFile) LogFatal(format string, args ...interface{}) {
	if c.level > XLogLevelFatal {
		return
	}
	c.writeToChan(XLogLevelFatal, c.module, format, args...)
}

func (c *XFile) SetLevel(level int) {
	c.level = level
}

func (c *XFile)Close()  {
	//管道为空要关闭
	if c.logChan != nil {
		close(c.logChan)
	}
	c.wg.Wait()
	if c.file != nil {
		c.file.Sync()  //同步写磁盘
		c.file.Close()
	}
}