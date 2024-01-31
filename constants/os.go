package constants

const (
	// ExitCode is the exit code used for a graceful exit of the program.
	ExitCode = 0

	// ExitCodeError is the generic exit code used when the program exits in an errored state.
	ExitCodeError = 1

	// ExitCodeSigint is the exit code used when the program is interrupted by a SIGINT/SIGTERM.
	ExitCodeSigint = 130

	// PermissionsEveryoneAllPermissions is 0777 permissions for files/directories -- everyone has
	// read, write, and execute permissions.
	PermissionsEveryoneAllPermissions = 0o777

	// PermissionsEveryoneReadWriteOwnerExecute is 0755 permissions for files/directories --
	// everyone can read, and execute, and owner can write.
	PermissionsEveryoneReadWriteOwnerExecute = 0o755

	// PermissionsEveryoneReadWrite is 0666 permissions for files/directories -- everyone has read
	// and write permissions.
	PermissionsEveryoneReadWrite = 0o666

	// PermissionsEveryoneReadExecute is 0555 permissions for files/directories -- everyone has read
	// and execute permissions.
	PermissionsEveryoneReadExecute = 0o555

	// PermissionsEveryoneRead is 0444 permissions for files/directories -- everyone has read
	// permissions.
	PermissionsEveryoneRead = 0o444
)
