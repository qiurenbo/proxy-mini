package common

import (
	"io"
	"os"
	"path"
	"time"

	runtime "github.com/banzaicloud/logrus-runtime-formatter"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
)

var (
	logFilePath = "./logs"
	logFileName = "verbose"
)

func InitLogger() *logrus.Logger {

	os.Mkdir("logs", 0777)

	// 日志文件
	fileName := path.Join(logFilePath, logFileName)

	// 写入文件
	// ! https://github.com/sirupsen/logrus/issues/1084
	f, _ := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	// 实例化
	logger := logrus.New()

	// https://bglobal.Logger.csdn.net/qmhball/article/details/116656368
	logrus.SetReportCaller(true)

	// 输出行数 https://github.com/sirupsen/logrus/issues/843
	formatter := runtime.Formatter{ChildFormatter: &logrus.TextFormatter{
		ForceQuote:      true,                  //键值对加引号
		TimestampFormat: "2006-01-02 15:04:05", //时间格式
		FullTimestamp:   true,
		ForceColors:     true}}
	formatter.Line = true

	// 设置输出格式 https://cloud.tencent.com/developer/article/1830707
	logger.SetFormatter(&formatter)

	// 即输出到标准输出又输入到文件 https://bglobal.Logger.csdn.net/XiaoWhy/article/details/107209317
	writers := []io.Writer{
		f,
		os.Stdout}
	//同时写文件和屏幕
	fileAndStdoutWriter := io.MultiWriter(writers...)

	logger.SetOutput(fileAndStdoutWriter)

	//设置日志级别
	logger.SetLevel(logrus.DebugLevel)

	// 设置 rotatelogs
	logWriter, _ := rotatelogs.New(
		// 分割后的文件名称
		fileName+".%Y%m%d.log",

		// 生成软链，指向最新日志文件
		// rotatelogs.WithLinkName(fileName),

		// 设置最大保存时间(7天)
		rotatelogs.WithMaxAge(7*24*time.Hour),

		// 设置日志切割时间间隔(1天)
		rotatelogs.WithRotationTime(24*time.Hour),
	)

	writeMap := lfshook.WriterMap{
		logrus.InfoLevel:  logWriter,
		logrus.FatalLevel: logWriter,
		logrus.DebugLevel: logWriter,
		logrus.WarnLevel:  logWriter,
		logrus.ErrorLevel: logWriter,
		logrus.PanicLevel: logWriter,
	}

	logger.AddHook(lfshook.NewHook(writeMap, &logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	}))

	return logger
}
