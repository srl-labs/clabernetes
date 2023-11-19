package launcher

import (
	"bytes"
	"fmt"
	"os/exec"
)

func (c *clabernetes) handleMounts() error {
	err := c.handleRemounts()
	if err != nil {
		return err
	}

	isCgroupV2, err := c.isCgroupV2()
	if err != nil {
		return err
	}

	if isCgroupV2 {
		// w/ cgroupv2 we'll have already remounted /sys/fs/cgroup as rw which should be all we
		// need to do; for cgroupv1 things we need to continue on to remount the sub components
		c.logger.Debug("running cgroupv2, no more remounting to do...")

		return nil
	}

	c.logger.Debug("handling cgroupv1 remounts...")

	err = c.handleCgroupV1Remounts()
	if err != nil {
		return err
	}

	return nil
}

func (c *clabernetes) handleRemounts() error {
	for _, remountPath := range []string{
		"/sys/fs/cgroup",
		"/proc",
		"/proc/sys",
	} {
		updateCmd := exec.Command( //nolint:gosec
			"mount",
			"-v",
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

func (c *clabernetes) handleCgroupV1Remounts() error {
	cgroupRemounts := []string{
		"blkio",
		"cpu,cpuacct",
		"cpuset",
		"devices",
		"freezer",
		"hugetlb",
		"memory",
		"net_cls,net_prio",
		"perf_event",
		"pids",
		"rdma",
	}

	for _, path := range cgroupRemounts {
		updateCmd := exec.Command( //nolint:gosec
			"umount",
			fmt.Sprintf("/sys/fs/cgroup/%s", path),
		)

		updateCmd.Stdout = c.logger
		updateCmd.Stderr = c.logger

		err := updateCmd.Run()
		if err != nil {
			return err
		}
	}

	for _, path := range cgroupRemounts {
		updateCmd := exec.Command( //nolint:gosec
			"mount",
			"cgroup",
			"-v",
			"-t",
			"cgroup",
			fmt.Sprintf("/sys/fs/cgroup/%s", path),
			"-o",
			fmt.Sprintf("%s,rw", path),
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

func (c *clabernetes) isCgroupV2() (bool, error) {
	// exec via bash to be lazy about piping :)
	checkCgroupMountVersionCmd := exec.Command(
		"bash",
		"-c",
		"mount | grep /sys/fs/cgroup",
	)

	checkCgroupMountVersionCmd.Stderr = c.logger

	outBytes, err := checkCgroupMountVersionCmd.Output()
	if err != nil {
		return false, err
	}

	if bytes.Contains(outBytes, []byte("cgroup2")) {
		return true, nil
	}

	return false, nil
}
