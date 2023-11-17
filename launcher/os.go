package launcher

import "os/exec"

func (c *clabernetes) handleRemounts() error {
	for _, remountPath := range []string{
		"/sys/fs/cgroup",
		"/proc",
		"/proc/sys",
	} {
		updateCmd := exec.Command( //nolint:gosec
			"mount",
			"-o",
			"remount,rw",
			remountPath,
			remountPath,
		)

		updateCmd.Stdout = c.logger
		updateCmd.Stderr = c.logger

		err := updateCmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}
