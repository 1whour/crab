package slog

import (
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog"
	//"github.com/rs/zerolog/log"
)

type Slog struct {
	zerolog.Logger
}

var once sync.Once

// 初始化函数
func New(w ...io.Writer) *Slog {

	once.Do(func() {

		zerolog.TimeFieldFormat = time.RFC3339Nano
	})
	return &Slog{Logger: zerolog.New(io.MultiWriter(w...)).With().Timestamp().Logger()}
}

// 设置日志等级
func (s *Slog) SetLevel(level string) *Slog {
	l, err := zerolog.ParseLevel(level)
	if err != nil {
		panic(err)
	}

	s.Logger = s.Level(l)
	return s
}

// 设置字段，一般是进程初始化级别才需要设置
func (s *Slog) Str(key, val string) *Slog {
	s.Logger = s.Logger.With().Str(key, val).Logger()
	return s
}

func (s *Slog) Debug() *event {
	return &event{s.Logger.Debug()}
}

func (s *Slog) Info() *event {
	return &event{s.Logger.Info()}
}

func (s *Slog) Warn() *event {
	return &event{s.Logger.Warn()}
}

func (s *Slog) Error(skip ...int) *event {
	nskip := 1
	if len(skip) > 0 {
		nskip = skip[0] + 1
	}

	_, file, line, ok := runtime.Caller(nskip)
	if !ok {
		return &event{s.Logger.Error()}
	}

	return &event{s.Logger.Error().Str("stack", fmt.Sprintf("%s:%d", file, line))}
}

type event struct {
	*zerolog.Event
}

func (e *event) ID(id string) *event {
	return &event{e.Str("ID", id)}
}

func (e *event) IP(ip string) *event {
	return &event{e.Str("IP", ip)}
}
