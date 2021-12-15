package graylogger

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net"
	"strings"
	"time"
)

type GraylogHook struct {
	conn               net.Conn
	graylogServAddress string
	hostToLogAs        string
	Level              logrus.Level
	numRetries         int
}

func NewGraylogger(address string, hostToLogAs string) *logrus.Logger {
	logger := logrus.New()
	appGuid := strings.Split(uuid.New().String(), "-")[0]
	logger.SetLevel(logrus.DebugLevel)
	logger.AddHook(NewGraylogHook(address, hostToLogAs+"-"+appGuid))
	return logger
}

func NewGraylogHook(address string, hostToLogAs string) *GraylogHook {
	return &GraylogHook{
		hostToLogAs:        hostToLogAs,
		graylogServAddress: address,
		numRetries:         3,
		Level:              logrus.DebugLevel,
	}
}

func (g *GraylogHook) SetLevel(level string) error {
	switch strings.ToLower(level) {
	case "panic":
		g.Level = logrus.PanicLevel
	case "fatal":
		g.Level = logrus.FatalLevel
	case "error":
		g.Level = logrus.ErrorLevel
	case "warn":
		g.Level = logrus.WarnLevel
	case "info":
		g.Level = logrus.InfoLevel
	case "debug":
		g.Level = logrus.DebugLevel
	case "trace":
		g.Level = logrus.TraceLevel
	default:
		return fmt.Errorf("[logger]: invalid log level passed: %s", level)
	}
	return nil
}

func (g *GraylogHook) Fire(entry *logrus.Entry) error {
	logData := make(logrus.Fields)
	logData["host"] = g.hostToLogAs
	logData["version"] = "1.1"
	logData["level"] = entry.Level
	logData["short_message"] = entry.Message
	logData["timestamp"] = entry.Time.Unix()
	for key, value := range entry.Data {
		logData[fmt.Sprintf("_%s", key)] = value
	}

	messageBytes, err := json.Marshal(logData)
	if err != nil {
		fmt.Printf("logger: error serializing log to json: %v", err)
		return fmt.Errorf("logger: error serializing log to json: %w", err)
	}

	if err := g.sendData(messageBytes); err != nil {
		fmt.Printf("logger: error sending log to graylog: %v", err)
		return fmt.Errorf("logger: error sending log to graylog: %w", err)
	}

	return nil
}

func (g *GraylogHook) connect() error {
	if g.conn != nil {
		return nil
	}

	var err error
	for i := 0; i < g.numRetries; i++ {
		if g.conn != nil { // close and remove any existing connection
			g.conn.Close()
			g.conn = nil
		}

		g.conn, err = net.Dial("tcp", g.graylogServAddress)
		if err != nil { // on error, let's do some retries
			time.Sleep(200 * time.Millisecond)
			continue
		}
		break // connected with no error; we're done
	}

	if err != nil { // return an error if we couldn't get the connection
		return fmt.Errorf("logger: error connecting to graylog after %d attempts: %w", g.numRetries, err)
	}
	return nil

}

func (g *GraylogHook) Levels() []logrus.Level {
	var levels []logrus.Level
	for _, level := range logrus.AllLevels {
		if level <= g.Level {
			levels = append(levels, level)
		}
	}
	return levels
}

func (g *GraylogHook) sendData(messageBytes []byte) error {
	messageBytes = append(messageBytes, byte(0)) // null byte delimits frame for graylog GELF over TCP

	var err error
	for i := 0; i < g.numRetries; i++ {
		g.connect() // Check that connection is active

		_, err := g.conn.Write(messageBytes)
		if err != nil { // close and reset connection before next retry
			g.conn.Close()
			g.conn = nil
			continue
		}
		break // message sent with no error; we're done.
	}

	if err != nil { // return the error if we couldn't send after retries
		return fmt.Errorf("logger: error sending log to graylog after %d attempts : %w", g.numRetries, err)
	}
	return nil
}
