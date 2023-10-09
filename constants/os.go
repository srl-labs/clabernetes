package constants

const (
	// ExitCode is the exit code used for a graceful exit of the program.
	ExitCode = 0

	// ExitCodeError is the generic exit code used when the program exits in an errored state.
	ExitCodeError = 1

	// ExitCodeSigint is the exit code used when the program is interrupted by a SIGINT/SIGTERM.
	ExitCodeSigint = 130

	// PermissionsEveryoneRead is 0755 permissions for files/directories -- everyone can read, and
	// execute, and owner can write.
	PermissionsEveryoneRead = 0o755

	// PermissionsEveryoneReadWrite is 0666 permissions for files/directories -- everyone has read
	// and write permissions.
	PermissionsEveryoneReadWrite = 0o666
)
