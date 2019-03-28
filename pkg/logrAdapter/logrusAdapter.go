package logrAdapter

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
)

func NewLogrusAdapter(l logrus.FieldLogger) logr.Logger {
	return &logrusLogf{logger: l}
}

type logrusLogf struct {
	logger logrus.FieldLogger
}

func (l logrusLogf) Info(msg string, keysAndValues ...interface{}) {
	l.WithValues(keysAndValues...).(logrusLogf).logger.Info(msg)
}

func (l logrusLogf) Enabled() bool {
	return true
}

func (l logrusLogf) Error(err error, msg string, keysAndValues ...interface{}) {
	o := l.WithValues(keysAndValues...).(logrusLogf)
	o.logger.WithError(err).Error(msg)
}

func (l logrusLogf) V(level int) logr.InfoLogger {
	return l
}

func (l logrusLogf) WithValues(keysAndValues ...interface{}) logr.Logger {
	olog := l.logger
	for i := 0; i < len(keysAndValues); i += 2 {
		olog = olog.WithField(fmt.Sprint(keysAndValues[i]), keysAndValues[i+1])
	}
	return logrusLogf{olog}
}

func (l logrusLogf) WithName(name string) logr.Logger {
	return logrusLogf{l.logger.WithField("name", name)}
}
