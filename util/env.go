package util

import (
	"os"
	"strconv"
)

// GetEnvStrOrDefault returns the value of the environment variable k as a string *or* the default d
// if casting fails or the environment variable is not set.
func GetEnvStrOrDefault(k, d string) string {
	v, ok := os.LookupEnv(k)
	if ok && v != "" {
		return v
	}

	return d
}

// GetEnvIntOrDefault returns the value of the environment variable k as an int *or* the default d
// if casting fails or the environment variable is not set.
func GetEnvIntOrDefault(k string, d int) int {
	v, ok := os.LookupEnv(k)
	if ok {
		ev, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return d
		}

		return int(ev)
	}

	return d
}

// GetEnvFloat64OrDefault returns the value of the environment variable k as a float *or* the
// default d if casting fails or the environment variable is not set.
func GetEnvFloat64OrDefault(k string, d float64) float64 {
	v, ok := os.LookupEnv(k)
	if ok {
		ev, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return d
		}

		return ev
	}

	return d
}

// GetEnvBoolOrDefault returns true if the environment variable is set, or the default d if it is
// unset.
func GetEnvBoolOrDefault(k string, d bool) bool {
	_, ok := os.LookupEnv(k)
	if ok {
		return true
	}

	return d
}
