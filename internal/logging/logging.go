package logging

import (
	"log"
	"os"
)

type LogCategory string

const (
	LogInfo  LogCategory = "INFO"
	LogDebug LogCategory = "DEBUG"
	LogError LogCategory = "ERROR"
	LogFatal LogCategory = "FATAL"
)

var logStream = make(chan string, 32)

func LogMessage(lc LogCategory, msg string) {
	if logStream == nil {
		log.Fatalf("%s - Log stream not initialised, InitLogging() has not been called")
		return
	}
	logStream <- string(lc) + " - " + msg
}

func InitLogging(logPath string) {
	logToFile := false
	var f *os.File
	if logPath != "" {
		logToFile = true
		f, _ = os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		defer f.Close()
		log.SetOutput(f)
	}
	for {
		select {
		case msg, ok := <-logStream:
			if !ok {
				log.Fatalf("%s - Log stream channel was killed, exiting", LogFatal)
				return
			}
			log.Println(msg)
			if logToFile {
				f.Sync()
			}
			switch msg {
			case string(LogFatal):
				os.Exit(1)
			}
		}
	}
}
