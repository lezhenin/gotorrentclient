package torrent

import (
	"github.com/sirupsen/logrus"
	"io"
)

type LoggerType int

const (
	SeederLogger  LoggerType = 0
	TrackerLogger LoggerType = 1
	ManagerLogger LoggerType = 2
	AllLoggers    LoggerType = 3
)

type LoggerLevel logrus.Level

const (
	PanicLevel   LoggerLevel = LoggerLevel(logrus.PanicLevel)
	ErrorLevel   LoggerLevel = LoggerLevel(logrus.ErrorLevel)
	WarningLevel LoggerLevel = LoggerLevel(logrus.WarnLevel)
	InfoLevel    LoggerLevel = LoggerLevel(logrus.InfoLevel)
	DebugLevel   LoggerLevel = LoggerLevel(logrus.DebugLevel)
	TraceLevel   LoggerLevel = LoggerLevel(logrus.TraceLevel)
)

var logsDir = "./logs"

var seederLogger = logrus.New()
var trackerLogger = logrus.New()
var managerLogger = logrus.New()

var loggers map[LoggerType]*logrus.Logger

func init() {

	loggers = make(map[LoggerType]*logrus.Logger)
	loggers[SeederLogger] = seederLogger
	loggers[TrackerLogger] = trackerLogger
	loggers[ManagerLogger] = managerLogger

	//file, err := os.Create("seeder.log")
	//if err == nil {
	//	SetLoggerOutput(SeederLogger, file)
	//} else {
	//	logrus.Error(errors.Annotate(err, "init torrent package"))
	//}

	seederLogger.SetLevel(logrus.TraceLevel)
	trackerLogger.SetLevel(logrus.TraceLevel)
	managerLogger.SetLevel(logrus.TraceLevel)

}

func SetLoggerOutput(loggerType LoggerType, writer io.Writer) {

	if loggerType == AllLoggers {
		for _, v := range loggers {
			v.SetOutput(writer)
		}
	} else {
		logger := loggers[loggerType]
		logger.SetOutput(writer)
	}
}

func SetLoggerLevel(loggerType LoggerType, level LoggerLevel) {

	if loggerType == AllLoggers {
		for _, v := range loggers {
			v.SetLevel(logrus.Level(level))
		}
		logrus.SetLevel(logrus.Level(level))
	} else {
		logger := loggers[loggerType]
		logger.SetLevel(logrus.Level(level))
	}
}
