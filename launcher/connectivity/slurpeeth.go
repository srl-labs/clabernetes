//go:build linux
// +build linux

package connectivity

import (
	"fmt"
	"os"
	"time"

	"github.com/carlmontanari/slurpeeth/slurpeeth"
	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerywatch "k8s.io/apimachinery/pkg/watch"
)

const (
	slurpeethConfigPath  = "/clabernetes/slurpeeth.yaml"
	slurpeethDialTimeout = 5 * time.Minute
)

type slurpeethManager struct {
	*common
	cancelChan chan bool
}

func (m *slurpeethManager) Run() {
	m.logger.Info(
		"containerlab started, connectivity mode is 'slurpeeth', initializing slurpeeth manager...",
	)

	m.renderSlurpeethConfig(m.initialTunnels)

	sm, err := slurpeeth.GetManager(
		slurpeeth.WithConfigFile(slurpeethConfigPath),
		slurpeeth.WithLiveReload(true),
		// timeout is really big for now because there may be weird delays while waiting for images
		// to pull/containers to schedule... maybe we want to re-think even setting a timeout and
		// just let it try over and over and over again
		slurpeeth.WithDialTimeout(slurpeethDialTimeout),
		// *probably* we also want to retry if this fails... not sure yet, so we'll try this and see
		// how it feels
		slurpeeth.WithWorkerRetry(true),
	)
	if err != nil {
		m.logger.Fatalf(
			"failed creating slurpeeth manager, error: %s",
			err,
		)
	}

	exitErr := make(chan bool)
	exitDone := make(chan bool)

	err = sm.RunDaemon(exitErr, exitDone)
	if err != nil {
		m.logger.Fatalf(
			"failed starting slurpeeth, error: %s",
			err,
		)
	}

	// watch the exit channels for slurpeeth in background, if they exit we can signal to the main
	// clabernetes process to bail out
	go func() {
		select {
		case <-exitDone:
			m.logger.Warn(
				"received exit signal from slurpeeth (non-error), sending done signal",
			)

			m.cancelChan <- true

			return
		case <-exitErr:
			m.logger.Critical(
				"received exit signal from slurpeeth (error), sending done signal",
			)

			m.cancelChan <- true

			return
		}
	}()

	m.logger.Debug("start connectivity custom resource watch...")

	go m.watchConnectivity()

	m.logger.Debug("slurpeeth connectivity setup complete")
}

func (m *slurpeethManager) watchConnectivity() {
	nodeName := os.Getenv(clabernetesconstants.LauncherNodeNameEnv)

	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", os.Getenv(
			clabernetesconstants.LauncherTopologyNameEnv,
		)),
		Watch: true,
	}

	watch, err := m.clabernetesClient.ClabernetesV1alpha1().
		Connectivities(os.Getenv(clabernetesconstants.PodNamespaceEnv)).
		Watch(m.ctx, listOptions)
	if err != nil {
		m.logger.Fatalf("failed watching clabernetes connectivity, err: %s", err)
	}

	for event := range watch.ResultChan() {
		switch event.Type { //nolint:exhaustive
		case apimachinerywatch.Modified:
			m.logger.Info("processing connectivity modification event")

			tunnelsCR, ok := event.Object.(*clabernetesapisv1alpha1.Connectivity)
			if !ok {
				m.logger.Warn(
					"failed casting event object to connectivity custom resource," +
						" this is probably a bug",
				)

				continue
			}

			nodeTunnels, ok := tunnelsCR.Spec.PointToPointTunnels[nodeName]
			if !ok {
				m.logger.Warnf(
					"no tunnels found for node %q, continuing but things may be broken",
					nodeName,
				)
			}

			m.renderSlurpeethConfig(nodeTunnels)
		default:
			m.logger.Warnf(
				"connectivity resource had %s event occur, ignoring...", event.Type,
			)
		}
	}
}

func (m *slurpeethManager) renderSlurpeethConfig(
	tunnels []*clabernetesapisv1alpha1.PointToPointTunnel,
) {
	slurpeethConfig := slurpeeth.Config{}

	for _, tunnel := range tunnels {
		slurpeethConfig.Segments = append(
			slurpeethConfig.Segments,
			slurpeeth.Segment{
				Name: fmt.Sprintf(
					"%s -> %s/%s",
					tunnel.LocalInterface,
					tunnel.RemoteNode,
					tunnel.RemoteInterface,
				),
				ID: uint16(tunnel.TunnelID),
				Interfaces: []string{
					fmt.Sprintf("%s-%s", tunnel.LocalNode, tunnel.LocalInterface),
				},
				Destinations: []string{tunnel.Destination},
			},
		)
	}

	slurpeethConfigYAML, err := yaml.Marshal(slurpeethConfig)
	if err != nil {
		m.logger.Fatalf(
			"failed marshalling slurpeeth config, error: %s",
			err,
		)
	}

	err = os.WriteFile(
		slurpeethConfigPath,
		slurpeethConfigYAML,
		clabernetesconstants.PermissionsEveryoneReadWriteOwnerExecute,
	)
	if err != nil {
		m.logger.Fatalf(
			"failed writing slurpeeth config to disk, error: %s",
			err,
		)
	}
}
