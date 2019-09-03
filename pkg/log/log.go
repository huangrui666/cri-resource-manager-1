// Copyright 2019 Intel Corporation. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Level is the log message severity level below which we suppress messages.
type Level int32

const (
	// LevelDebug corresponds to debug messages.
	LevelDebug Level = iota
	// LevelInfo corresponds to informational messages.
	LevelInfo
	// LevelWarn corresponds to warning messages.
	LevelWarn
	// LevelError corresponds to error messages.
	LevelError
)

// Logger is the interface for configuring and producing log messages.
type Logger interface {
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
	Panic(format string, args ...interface{})

	DebugEnabled() bool
	Debug(format string, args ...interface{})

	Stop()
}

// Backend is an entity that can emit log messages.
type Backend interface {
	Name() string
	PrefixPreference() bool
	Enabled(Level) bool
	Info(message string)
	Warn(message string)
	Error(message string)

	Debug(message string)
}

// Our logger instance.
type logger struct {
	source  string // logger source/module name
	enabled bool   // logger source module
	level   Level  // first non-suppressed severity level
	debug   bool   // debugging for this instance
}

// Get an existing logger or create a new one.
func Get(source string) Logger {
	l, ok := opt.loggers[source]
	if !ok {
		return newLogger(source)
	}
	return l
}

// NewLogger creates a new logger, getting the existing one if possible.
func NewLogger(source string) Logger {
	return Get(source)
}

// newLogger creates a new logger instance.
func newLogger(source string) Logger {
	source = strings.Trim(source, "[] ")

	if opt.loggers == nil {
		opt.loggers = make(map[string]*logger)
	}

	if l := opt.loggers[source]; l != nil {
		return l
	}

	l := &logger{
		source:  source,
		enabled: opt.sourceEnabled(source),
		debug:   opt.debugEnabled(source),
		level:   opt.level,
	}
	opt.loggers[source] = l

	if opt.active == nil {
		SelectBackend("")
	}

	return l
}

// Optional call to stop a logger once it is not needed any more.
func (l *logger) Stop() {
	l.enabled = false
	delete(opt.loggers, l.source)
}

func (level Level) String() string {
	if name, ok := LevelNames[level]; ok {
		return name
	}

	return fmt.Sprintf("<unknown log level %d>", level)
}

func (l *logger) shouldPrefix() bool {
	return opt.prefix == 1 || (opt.prefix == -1 && opt.active.PrefixPreference())
}

func (l *logger) passthrough(level Level) bool {
	return (l.enabled && l.level <= level) || (level == LevelDebug && l.debug)
}

func (l *logger) formatMessage(format string, args ...interface{}) string {
	if len(l.source) > opt.srcalign {
		opt.srcalign = len(l.source)
	}

	msg := ""

	if l.shouldPrefix() {
		suf := (opt.srcalign - len(l.source)) / 2
		pre := opt.srcalign - (len(l.source) + suf)
		msg = "[" + fmt.Sprintf("%-*s", pre, "") + l.source + fmt.Sprintf("%*s", suf, "") + "] "
	}

	return msg + fmt.Sprintf(format, args...)
}

// Emit an info message (lowest priority).
func (l *logger) Info(format string, args ...interface{}) {
	if !l.passthrough(LevelInfo) {
		return
	}
	opt.active.Info(l.formatMessage(format, args...))
}

// Emit a warning message.
func (l *logger) Warn(format string, args ...interface{}) {
	if !l.passthrough(LevelWarn) {
		return
	}
	opt.active.Warn(l.formatMessage(format, args...))
}

// Emit an error message.
func (l *logger) Error(format string, args ...interface{}) {
	if !l.passthrough(LevelError) {
		return
	}
	opt.active.Error(l.formatMessage(format, args...))
}

// Emit a fatal error message and exit.
func (l *logger) Fatal(format string, args ...interface{}) {
	opt.active.Error(l.formatMessage(format, args...))
	os.Exit(1)
}

// Emit a fatal error message and panic.
func (l *logger) Panic(format string, args ...interface{}) {
	message := l.formatMessage(format, args...)
	opt.active.Error(message)
	panic(message)
}

// Default logger/source.
var defLogger Logger

// Get the default logger.
func Default() Logger {
	return defLogger
}

// Emit an info message with the default soruce.
func Info(format string, args ...interface{}) {
	defLogger.Info(format, args...)
}

// Emit a warning message with the default source.
func Warn(format string, args ...interface{}) {
	defLogger.Warn(format, args...)
}

// Emit an error message with the default source.
func Error(format string, args ...interface{}) {
	defLogger.Error(format, args...)
}

// Emit a fatal error message with the default source.
func Fatal(format string, args ...interface{}) {
	defLogger.Fatal(format, args...)
}

// Emit a fatal error message with the default source and panic.
func Panic(format string, args ...interface{}) {
	defLogger.Panic(format, args...)
}

// Emit a debug message with the default source.
func Debug(format string, args ...interface{}) {
	defLogger.Debug(format, args...)
}

// Check if debugging is enabled.
func (l *logger) DebugEnabled() bool {
	return l.debug
}

// Emit a debug message.
func (l *logger) Debug(format string, args ...interface{}) {
	if !l.debug {
		return
	}
	opt.active.Debug(l.formatMessage(format, args...))
}

// RegisterBackend registers a logger backend.
func RegisterBackend(b Backend) {
	name := b.Name()

	if opt.backends == nil {
		opt.backends = make(map[string]Backend)
	}

	opt.backends[name] = b

	if opt.logger == name {
		opt.active = b
	}
}

// SelectBackend selects the logger backend to activate.
func SelectBackend(name string) {
	if name != "" {
		name = opt.logger
	}

	if b, ok := opt.backends[name]; ok {
		opt.active = b
	} else {
		for _, name := range defaultBackends {
			if b, ok := opt.backends[name]; ok {
				opt.active = b
				break
			}
		}
	}

	if opt.active != nil {
		opt.active.Info(fmt.Sprintf("activated logger backend '%s'",
			opt.active.Name()))
	}
}

// Get the names of registered backends.
func registeredBackendNames() string {
	names := ""
	sep := ""
	for name := range opt.backends {
		names += sep + name
		sep = ","
	}
	return names
}

// Update loggers when debug flags or sources change.
func (o *options) updateLoggers() {
	for s, l := range o.loggers {
		l.enabled = opt.sourceEnabled(s)
		l.debug = opt.debugEnabled(s)
		l.level = opt.level
	}
}

//
// fallback fmt backend, using fmt.*Printf
//

type fmtBackend struct{}

var _ Backend = &fmtBackend{}

func (f *fmtBackend) Name() string {
	return "fmt"
}

func (f *fmtBackend) PrefixPreference() bool {
	return true
}

func (f *fmtBackend) Info(message string) {
	fmt.Println("I: " + message)
}

func (f *fmtBackend) Warn(message string) {
	fmt.Println("W: " + message)
}

func (f *fmtBackend) Error(message string) {
	fmt.Println("E: " + message)
}

func (f *fmtBackend) Debug(message string) {
	fmt.Println("D: " + message)
}

func (f *fmtBackend) Enabled(l Level) bool {
	return l >= opt.level
}

func init() {
	RegisterBackend(&fmtBackend{})

	binary := filepath.Clean(os.Args[0])
	source := filepath.Base(binary)
	defLogger = newLogger(source)
}
