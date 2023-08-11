package clabverter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	clabernetesutil "gitlab.com/carlmontanari/clabernetes/util"

	clabernetesutilcontainerlab "gitlab.com/carlmontanari/clabernetes/util/containerlab"

	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"
	claberneteslogging "gitlab.com/carlmontanari/clabernetes/logging"
)

const specIndentSpaces = 4

type configMapTemplateVars struct {
	Name          string
	Namespace     string
	StartupConfig string
}

type topologyConfigMapTemplateVars struct {
	NodeName      string
	ConfigMapName string
	FilePath      string
}

type containerlabTemplateVars struct {
	Name               string
	Namespace          string
	ClabConfig         string
	StartupConfigs     []topologyConfigMapTemplateVars
	InsecureRegistries []string
}

// MustNewClabverter returns an instance of Clabverter or panics.
func MustNewClabverter(
	topologyFile,
	outputDirectory,
	destinationNamespace,
	insecureRegistries string,
	debug,
	quiet,
	stdout bool,
) *Clabverter {
	logLevel := clabernetesconstants.Info

	if debug {
		logLevel = clabernetesconstants.Debug
	}

	if quiet {
		logLevel = clabernetesconstants.Disabled
	}

	logManager := claberneteslogging.GetManager()

	clabverterLogger := logManager.MustRegisterAndGetLogger(
		clabernetesconstants.Clabverter,
		logLevel,
	)

	return &Clabverter{
		logger:                  clabverterLogger,
		topologyFile:            topologyFile,
		outputDirectory:         outputDirectory,
		stdout:                  stdout,
		destinationNamespace:    destinationNamespace,
		insecureRegistries:      strings.Split(insecureRegistries, ","),
		startupConfigConfigMaps: make(map[string]topologyConfigMapTemplateVars),
	}
}

// Clabverter is a struct that holds data/methods for "clabversion" -- that is, the conversion of
// a "normal" containerlab topology to a clabernetes Containerlab resource, and any other associated
// manifest(s).
type Clabverter struct {
	logger claberneteslogging.Instance

	topologyFile    string
	outputDirectory string
	stdout          bool

	destinationNamespace string

	insecureRegistries []string

	topologyPath       string
	topologyPathParent string

	rawClabConfig string
	clabConfig    *clabernetesutilcontainerlab.Config

	// mapping of nodeName -> startup-config info for the templating process
	startupConfigConfigMaps map[string]topologyConfigMapTemplateVars
}

// Clabvert is the main (only) entrypoint that kicks off the "clabversion" process.
func (c *Clabverter) Clabvert() error {
	c.logger.Info("starting clabversion!")

	err := c.ensureOutputDirectory()
	if err != nil {
		return err
	}

	err = c.load()
	if err != nil {
		return err
	}

	err = c.handleStartupConfigs()
	if err != nil {
		return err
	}

	err = c.handleManifest()
	if err != nil {
		return err
	}

	c.logger.Info("clabversion complete!")

	return nil
}

func (c *Clabverter) ensureOutputDirectory() error {
	var err error

	c.outputDirectory, err = filepath.Abs(c.outputDirectory)
	if err != nil {
		c.logger.Criticalf("failed determining absolute path of output directory, error: %s", err)

		return err
	}

	err = os.MkdirAll(c.outputDirectory, clabernetesconstants.PermissionsEveryoneRead)
	if err != nil {
		c.logger.Criticalf("failed ensuring output directory exists, error: %s", err)

		return err
	}

	return nil
}

func (c *Clabverter) load() error {
	// get fully qualified path of the topo file
	var err error

	c.logger.Info("loading and validating provided containerlab topology file...")

	switch {
	case isURL(c.topologyFile):
		if strings.Contains(c.topologyFile, "github.com") &&
			!strings.Contains(c.topologyFile, "githubusercontent") {
			c.logger.Info("converting github link to raw style...")

			c.topologyFile = gitHubNormalToRawLink(c.topologyFile)
		}
	default:
		c.topologyPath, err = filepath.Abs(c.topologyFile)
		if err != nil {
			c.logger.Criticalf("failed determining absolute path of topology file, error: %s", err)

			return err
		}
	}

	c.logger.Debugf(
		"determined fully qualified containerlab topology path as: %s", c.topologyPath,
	)

	// make sure we set working dir to the dir of the topo file, or the "parent" folder if its a url
	switch {
	case isURL(c.topologyFile):
		pathParts := strings.Split(c.topologyFile, "/")

		c.topologyPathParent = strings.Join(pathParts[:len(pathParts)-1], "/")
	default:
		c.topologyPathParent = filepath.Dir(c.topologyPath)
	}

	c.logger.Debug("attempting to load containerlab topology....")

	var rawClabConfigBytes []byte

	switch {
	case isURL(c.topologyFile):
		rawClabConfigBytes, err = loadContentAtURL(c.topologyFile)
	default:
		rawClabConfigBytes, err = os.ReadFile(c.topologyFile)
	}

	if err != nil {
		c.logger.Criticalf(
			"failed reading containerlab topology file at '%s' from disk, error: %s",
			c.topologyPath, err,
		)

		return err
	}

	c.rawClabConfig = string(rawClabConfigBytes)

	// parse the topo file
	c.clabConfig, err = clabernetesutilcontainerlab.LoadContainerlabConfig(c.rawClabConfig)
	if err != nil {
		c.logger.Criticalf(
			"failed parsing containerlab topology file at '%s', error: %s", c.topologyPath, err,
		)

		return err
	}

	if len(c.clabConfig.Topology.Nodes) == 0 {
		c.logger.Info("no nodes in topology file, nothing to do...")

		return nil
	}

	c.logger.Debug("loading and validating containerlab topology file complete!")

	return nil
}

func (c *Clabverter) handleStartupConfigs() error {
	c.logger.Info("handling containerlab topology startup config(s) if present...")

	// determine if any startup configs exist (probably other stuff but at least this?)
	startupConfigs := map[string][]byte{}

	for nodeName, nodeData := range c.clabConfig.Topology.Nodes {
		if nodeData.StartupConfig == "" {
			c.logger.Debugf("node '%s' does not have a startup-config, skipping....", nodeName)

			continue
		}

		c.logger.Debugf("loading node '%s' startup-config...", nodeName)

		var startupConfigContents []byte

		var err error

		switch {
		case isURL(c.topologyFile):
			startupConfigContents, err = loadContentAtURL(
				fmt.Sprintf("%s/%s", c.topologyPathParent, nodeData.StartupConfig),
			)
		default:
			fullyQualifiedStartupConfigPath := fmt.Sprintf(
				"%s/%s",
				c.topologyPathParent,
				nodeData.StartupConfig,
			)

			startupConfigContents, err = os.ReadFile(fullyQualifiedStartupConfigPath) //nolint:gosec
		}

		if err != nil {
			c.logger.Criticalf(
				"failed loading startup-config contents for node '%s', error: %s",
				nodeName,
				err,
			)

			return err
		}

		startupConfigs[nodeName] = startupConfigContents
	}

	c.logger.Info("rendering clabernetes startup config outputs...")

	// render configmap(s) for startup configs
	for nodeName, startupConfigContents := range startupConfigs {
		t, err := template.ParseFS(Assets, "assets/startup-config-configmap.yaml.template")
		if err != nil {
			c.logger.Criticalf("failed loading configmap template from assets: %s", err)

			return err
		}

		configMapName := fmt.Sprintf("%s-%s-startup-config", c.clabConfig.Name, nodeName)

		var rendered bytes.Buffer

		err = t.Execute(
			&rendered,
			configMapTemplateVars{
				Name:      configMapName,
				Namespace: c.destinationNamespace,
				// pad w/ a newline so the template can look prettier :)
				StartupConfig: "\n" + clabernetesutil.Indent(
					string(startupConfigContents),
					specIndentSpaces,
				),
			},
		)
		if err != nil {
			c.logger.Criticalf("failed executing configmap template: %s", err)

			return err
		}

		if c.stdout {
			_, err = os.Stdout.Write(rendered.Bytes())
			if err != nil {
				c.logger.Criticalf(
					"failed writing node '%s' startup config to stdout: %s", nodeName, err,
				)

				return err
			}
		} else {
			err = os.WriteFile(
				fmt.Sprintf("%s/%s-startup-config.yaml", c.outputDirectory, nodeName),
				rendered.Bytes(),
				clabernetesconstants.PermissionsEveryoneRead,
			)
			if err != nil {
				c.logger.Criticalf(
					"failed writing node '%s' startup config to output directory: %s",
					nodeName,
					err,
				)

				return err
			}
		}

		c.startupConfigConfigMaps[nodeName] = topologyConfigMapTemplateVars{
			NodeName:      nodeName,
			ConfigMapName: configMapName,
			FilePath:      c.clabConfig.Topology.Nodes[nodeName].StartupConfig,
		}
	}

	c.logger.Debug("handling startup configs complete")

	return nil
}

func (c *Clabverter) handleManifest() error {
	t, err := template.ParseFS(Assets, "assets/containerlab.yaml.template")
	if err != nil {
		c.logger.Criticalf("failed loading containerlab manifest from assets: %s", err)

		return err
	}

	startupConfigs := make([]topologyConfigMapTemplateVars, len(c.startupConfigConfigMaps))

	var idx int

	for _, startupConfig := range c.startupConfigConfigMaps {
		startupConfigs[idx] = startupConfig

		idx++
	}

	var rendered bytes.Buffer

	err = t.Execute(
		&rendered,
		containerlabTemplateVars{
			Name:      c.clabConfig.Name,
			Namespace: c.destinationNamespace,
			// pad w/ a newline so the template can look prettier :)
			ClabConfig:         "\n" + clabernetesutil.Indent(c.rawClabConfig, specIndentSpaces),
			StartupConfigs:     startupConfigs,
			InsecureRegistries: c.insecureRegistries,
		},
	)
	if err != nil {
		c.logger.Criticalf("failed executing configmap template: %s", err)

		return err
	}

	if c.stdout {
		_, err = os.Stdout.Write(rendered.Bytes())
		if err != nil {
			c.logger.Criticalf(
				"failed writing containerlab manifest to stdout: %s", err,
			)

			return err
		}
	} else {
		err = os.WriteFile(
			fmt.Sprintf("%s/%s.yaml", c.outputDirectory, c.clabConfig.Name),
			rendered.Bytes(),
			clabernetesconstants.PermissionsEveryoneRead,
		)
		if err != nil {
			c.logger.Criticalf(
				"failed writing containerlab manifest to output directory: %s", err,
			)

			return err
		}
	}

	return nil
}
