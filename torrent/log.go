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
	logger := loggers[loggerType]
	logger.SetOutput(writer)
}
