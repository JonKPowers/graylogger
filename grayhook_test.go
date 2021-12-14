package graylogger

import (
	"github.com/sirupsen/logrus"
	"testing"
)

func TestGrahookLoad(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.AddHook(NewGraylogHook("graylog.ladyjlabs.com:12201", "TestApp"))

	for i := 0; i < 100000; i++ {
		logger.WithFields(logrus.Fields{
			"TestField1": "Here's some stuff",
			"TestField2": "Here's some other stuff",
		}).Debugf("This is log message number %d", i)
	}

}
