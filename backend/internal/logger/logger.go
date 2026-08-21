package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
)

func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return Debug
	case "warn", "warning":
		return Warn
	case "error":
		return Error
	default:
		return Info
	}
}

func (l Level) String() string {
	switch l {
	case Debug:
		return "debug"
	case Warn:
		return "warn"
	case Error:
		return "error"
	default:
		return "info"
	}
}

type Logger struct {
	mu    sync.Mutex
	level Level
	out   io.Writer
	loc   *time.Location
}

func New(level Level, out io.Writer) *Logger {
	if out == nil {
		out = os.Stdout
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("GMT+8", 8*3600)
	}
	return &Logger{level: level, out: out, loc: loc}
}

func (l *Logger) Debug(msg string, kv ...any) { l.log(Debug, msg, kv...) }
func (l *Logger) Info(msg string, kv ...any)  { l.log(Info, msg, kv...) }
func (l *Logger) Warn(msg string, kv ...any)  { l.log(Warn, msg, kv...) }
func (l *Logger) Error(msg string, kv ...any) { l.log(Error, msg, kv...) }

func (l *Logger) log(lv Level, msg string, kv ...any) {
	if lv < l.level {
		return
	}
	var b strings.Builder
	now := time.Now().In(l.loc).Format("2006-01-02 15:04:05")
	b.WriteString("time=")
	b.WriteString(now)
	b.WriteString(" level=")
	b.WriteString(lv.String())
	b.WriteString(" msg=")
	b.WriteString(quote(msg))
	for i := 0; i+1 < len(kv); i += 2 {
		b.WriteByte(' ')
		b.WriteString(fmt.Sprint(kv[i]))
		b.WriteByte('=')
		b.WriteString(quote(fmt.Sprint(kv[i+1])))
	}
	b.WriteByte('\n')
	l.mu.Lock()
	line := b.String()
	l.mu.Unlock()
	_, _ = io.WriteString(l.out, line)
}

func quote(s string) string {
	if strings.ContainsAny(s, " \t=\"") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
