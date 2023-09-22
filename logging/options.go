package logging

// Option is a functional option type for applying options to the logging Manager.
type Option func(m *manager)

// WithLogger appends a logger (a function that accepts an interface) to the logging Manager's set
// of loggers.
func WithLogger(logger func(...interface{})) Option {
	return func(m *manager) {
		m.loggers = append(m.loggers, logger)
	}
}
