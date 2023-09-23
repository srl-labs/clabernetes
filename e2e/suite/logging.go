package suite

import "fmt"

const (
	green     = "\u001B[32m"
	red       = "\u001B[31m"
	colorStop = "\033[0m"
)

// LogStepDescr sends a pretty string to the test logger for a step starting.
func LogStepDescr(idx int, description string) string {
	return fmt.Sprintf("Step %d: %s", idx, description)
}

// LogStepSuccess sends a pretty string to the test logger for a step success.
func LogStepSuccess(idx int) string {
	return fmt.Sprintf("Step %d: %sSUCCESS%s", idx, green, colorStop)
}

// LogStepFailure sends a pretty string to the test logger for a step failure.
func LogStepFailure(idx int) string {
	return fmt.Sprintf("Step %d: %sFAILURE%s", idx, red, colorStop)
}
