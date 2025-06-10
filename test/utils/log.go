package utils

import (
	"bytes"

	"github.com/go-logr/logr"
)

type FakeLogger struct {
	logr.Logger
	Err  error
	Buff bytes.Buffer
}

func (l *FakeLogger) Init(info logr.RuntimeInfo) {}
func (l *FakeLogger) Enabled(lvl int) bool       { return true }
func (l *FakeLogger) Info(lvl int, msg string, keysAndValues ...interface{}) {
	l.Buff.WriteString(msg)
}
func (l *FakeLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.Buff.WriteString(msg)
	l.Err = err
}
func (l *FakeLogger) WithValues(keysAndValues ...interface{}) logr.LogSink { return l }
func (l *FakeLogger) WithName(name string) logr.LogSink                    { return l }
