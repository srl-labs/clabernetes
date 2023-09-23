package clabverter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	clabernetesutil "github.com/srl-labs/clabernetes/util"

	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
)

const specIndentSpaces = 4

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

	claberneteslogging.InitManager(
		claberneteslogging.WithLogger(claberneteslogging.StdErrLog),
	)

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
		extraFilesConfigMaps:    make(map[string][]topologyConfigMapTemplateVars),
		renderedFiles:           []renderedContent{},
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

	// mapping of nodeName -> startup-config info for the templating process; this is its own thing
	// because configurations may be huge and configmaps have a 1M char limit, so while keeping them
	// by themselves may not "solve" for ginormous configs, it can certainly give us a little extra
	// breathing room by not having other data in the startup configmap.
	startupConfigConfigMaps map[string]topologyConfigMapTemplateVars

	// all other config files associated to the node(s) -- for example license file(s).
	extraFilesConfigMaps map[string][]topologyConfigMapTemplateVars

	// filenames -> content of all rendered files we need to either print to stdout or write to disk
	renderedFiles []renderedContent
}

// Clabvert is the main (only) entrypoint that kicks off the "clabversion" process.
func (c *Clabverter) Clabvert() error {
	c.logger.Info("starting clabversion!")

	err := c.ensureOutputDirectory()
	if err != nil {
		return err
	}

	err = c.findClabTopologyFile()
	if err != nil {
		return err
	}

	err = c.load()
	if err != nil {
		return err
	}

	err = c.handleAssociatedFiles()
	if err != nil {
		return err
	}

	err = c.handleManifest()
	if err != nil {
		return err
	}

	err = c.output()
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

	err = os.MkdirAll(c.outputDirectory, clabernetesconstants.PermissionsEveryoneReadUserWrite)
	if err != nil {
		c.logger.Criticalf("failed ensuring output directory exists, error: %s", err)

		return err
	}

	return nil
}

func (c *Clabverter) resolveContentAtPath(path string) ([]byte, error) {
	var content []byte

	var err error

	switch {
	case isURL(c.topologyFile):
		content, err = loadContentAtURL(
			fmt.Sprintf("%s/%s", c.topologyPathParent, path),
		)
	default:
		fullyQualifiedStartupConfigPath := path

		if !strings.HasPrefix(path, c.topologyPathParent) {
			// we may have already set this while processing bind mounts, so don't blindly add the
			// parent path unless we need to!
			fullyQualifiedStartupConfigPath = fmt.Sprintf(
				"%s/%s",
				c.topologyPathParent,
				path,
			)
		}

		content, err = os.ReadFile(fullyQualifiedStartupConfigPath) //nolint:gosec
	}

	return content, err
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

func (c *Clabverter) handleAssociatedFiles() error {
	c.logger.Info("handling containerlab associated file(s) if present...")

	err := c.handleStartupConfigs()
	if err != nil {
		return err
	}

	err = c.handleExtraFiles()
	if err != nil {
		return err
	}

	c.logger.Debug("handling associated file(s) complete")

	return nil
}

func (c *Clabverter) handleManifest() error {
	t, err := template.ParseFS(Assets, "assets/containerlab.yaml.template")
	if err != nil {
		c.logger.Criticalf("failed loading containerlab manifest from assets: %s", err)

		return err
	}

	startupConfigs := make([]topologyConfigMapTemplateVars, len(c.startupConfigConfigMaps))

	var startupConfigsIdx int

	for _, startupConfig := range c.startupConfigConfigMaps {
		startupConfigs[startupConfigsIdx] = startupConfig

		startupConfigsIdx++
	}

	extraFiles := make([]topologyConfigMapTemplateVars, 0)

	for _, nodeExtraFiles := range c.extraFilesConfigMaps {
		extraFiles = append(extraFiles, nodeExtraFiles...)
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
			ExtraFiles:         extraFiles,
			InsecureRegistries: c.insecureRegistries,
		},
	)
	if err != nil {
		c.logger.Criticalf("failed executing configmap template: %s", err)

		return err
	}

	fileName := fmt.Sprintf("%s/%s.yaml", c.outputDirectory, c.clabConfig.Name)

	c.renderedFiles = append(
		c.renderedFiles,
		renderedContent{
			friendlyName: "clabernetes manifest",
			fileName:     fileName,
			content:      rendered.Bytes(),
		},
	)

	return nil
}

func (c *Clabverter) output() error {
	for _, rendered := range c.renderedFiles {
		if c.stdout {
			_, err := os.Stdout.Write(rendered.content)
			if err != nil {
				c.logger.Criticalf(
					"failed writing '%s' startup config to stdout: %s", rendered.friendlyName, err,
				)

				return err
			}
		} else {
			err := os.WriteFile(
				rendered.fileName,
				rendered.content,
				clabernetesconstants.PermissionsEveryoneRead,
			)
			if err != nil {
				c.logger.Criticalf(
					"failed writing '%s' to output directory: %s",
					rendered.friendlyName,
					err,
				)

				return err
			}
		}
	}

	return nil
}

// findClabTopologyFile attempts to find a clab file in the working directory if the path was not
// provided.
func (c *Clabverter) findClabTopologyFile() error {
	if c.topologyFile != "" {
		return nil
	}

	c.logger.Info("attempting to find topology file in the working directory...")

	files, err := filepath.Glob("*.clab.y*ml")
	if err != nil {
		return err
	}

	if len(files) != 1 {
		return fmt.Errorf(
			"%w: none or more than one topology files found, can't auto select one",
			ErrClabvert,
		)
	}

	c.logger.Infof("found topology file %q", files[0])

	c.topologyFile = files[0]

	return nil
}
