package sugar

import (
	"github.com/go-logr/logr"
)

// Sink
// TODO: 用于记录kubernetes component组件的日志包装器
// TODO: 底层我们还是使用sugarLogger进行记录
type Sink struct {
	level  string
	name   string
	values map[string]any
}

func (s *Sink) Init(info logr.RuntimeInfo) {
	println(1)
}

func (s *Sink) Enabled(level int) bool {
	println(2)
	return true
}

func (s *Sink) Info(level int, msg string, keysAndValues ...any) {
	println(3)
}

func (s *Sink) Error(err error, msg string, keysAndValues ...any) {
	println(4)
}

func (s *Sink) WithValues(keysAndValues ...any) logr.LogSink {
	println(5)
	return s
}

func (s *Sink) WithName(name string) logr.LogSink {
	println(6)
	return s
}
