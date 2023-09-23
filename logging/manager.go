package logging

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var (
	managerInstance     Manager   //nolint:gochecknoglobals
	managerInstanceOnce sync.Once //nolint:gochecknoglobals
)

const (
	flushWaitAttempts = 5
	flushWaitSleep    = 25 * time.Millisecond
)

// InitManager initializes the logging manager with the provided options. It does nothing if the
// manager has already been initialized. This may be a bit awkward but since this is not expected
// to be used by anything but clabernetes and it works for us, it works!
func InitManager(options ...Option) {
	managerInstanceOnce.Do(func() {
		m := &manager{
			formatter:       DefaultFormatter,
			instances:       map[string]*instance{},
			instanceCancels: map[string]context.CancelFunc{},
			loggers:         []func(...any){},
		}

		for _, option := range options {
			option(m)
		}

		if len(m.loggers) == 0 {
			m.loggers = []func(...interface{}){printLog}
		}

		managerInstance = m
	})
}

// GetManager returns the global logging Manager. Panics if the logging manager has *not* been
// initialized (via InitManager).
func GetManager() Manager {
	if managerInstance == nil {
		panic(
			"Manager instance is nil, 'GetManager' should never be called until the manager " +
				"process has been started",
		)
	}

	return managerInstance
}

// Manager is the interface representing the global logging manager singleton, this interface
// defines the ways to interact with this object.
type Manager interface {
	SetLoggerFormatter(name string, formatter Formatter) error
	SetLoggerFormatterAllInstances(formatter Formatter)
	SetLoggerLevelAllInstances(level string)
	SetLoggerLevel(name, level string) error
	RegisterLogger(name, level string) error
	RegisterAndGetLogger(name, level string) (Instance, error)
	MustRegisterAndGetLogger(name, level string) Instance
	GetLogger(name string) (Instance, error)
	DeleteLogger(name string)
	Flush()
}

// Manager is a "global" logging object for a clabernetes instance. It contains logging instances
// for individual components (controllers primarily), allowing logging at different levels for each
// component, but with a unified interface and message formatting system.
type manager struct {
	formatter       Formatter
	instances       map[string]*instance
	instanceCancels map[string]context.CancelFunc
	loggers         []func(...interface{})
}

func (m *manager) start(i *instance) {
	for {
		select {
		case <-i.done:
			// closing down, stop the goroutine
			return
		case logMsg := <-i.c:
			for _, f := range m.loggers {
				lf := f

				lf(logMsg)
			}
		}
	}
}

func (m *manager) flush(i *instance, wg *sync.WaitGroup) {
	var retryCount int

	for {
		select {
		case logMsg := <-i.c:
			retryCount = 0

			for _, f := range m.loggers {
				lf := f

				lf(logMsg)
			}
		default:
			if retryCount >= flushWaitAttempts {
				wg.Done()

				return
			}

			// the log messages may need a tiny bit more time to get slurped up in the channel for
			// each individual instance, so we sleep and count retries to delay things a tick
			time.Sleep(flushWaitSleep)

			retryCount++
		}
	}
}

func (m *manager) SetLoggerFormatterAllInstances(formatter Formatter) {
	for _, i := range m.instances {
		i.formatter = formatter
	}
}

func (m *manager) SetLoggerFormatter(name string, formatter Formatter) error {
	i, exists := m.instances[name]
	if !exists {
		return fmt.Errorf("%w: logger '%s' does not exist", ErrLoggingInstance, name)
	}

	i.formatter = formatter

	return nil
}

func (m *manager) SetLoggerLevelAllInstances(level string) {
	for _, i := range m.instances {
		i.level = level
	}
}

func (m *manager) SetLoggerLevel(name, level string) error {
	i, exists := m.instances[name]
	if !exists {
		return fmt.Errorf("%w: logger '%s' does not exist", ErrLoggingInstance, name)
	}

	i.level = level

	return nil
}

func (m *manager) RegisterLogger(name, level string) error {
	level, err := ValidateLogLevel(level)
	if err != nil {
		return err
	}

	_, exists := m.instances[name]
	if exists {
		return fmt.Errorf("%w: logger '%s' already exists", ErrLoggingInstance, name)
	}

	m.instances[name] = &instance{
		lock:      sync.Mutex{},
		name:      name,
		level:     level,
		formatter: m.formatter,
		c:         make(chan string),
		done:      make(chan interface{}),
	}

	go m.start(m.instances[name])

	return nil
}

func (m *manager) RegisterAndGetLogger(name, level string) (Instance, error) {
	err := m.RegisterLogger(name, level)
	if err != nil {
		return nil, err
	}

	i, err := m.GetLogger(name)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (m *manager) MustRegisterAndGetLogger(name, level string) Instance {
	i, err := m.RegisterAndGetLogger(name, level)
	if err != nil {
		panic(err)
	}

	return i
}

func (m *manager) GetLogger(name string) (Instance, error) {
	i, exists := m.instances[name]
	if !exists {
		return nil, fmt.Errorf(
			"%w: logger '%s' does not exist",
			ErrLoggingInstance,
			name,
		)
	}

	return i, nil
}

func (m *manager) DeleteLogger(name string) {
	delete(m.instances, name)
}

// Flush stops all loggers and emits all remaining messages.
func (m *manager) Flush() {
	wg := &sync.WaitGroup{}

	for _, i := range m.instances {
		wg.Add(1)

		m.flush(i, wg)

		close(i.done)
	}

	wg.Wait()
}
