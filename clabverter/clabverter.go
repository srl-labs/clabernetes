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

type startupConfigConfigMapTemplateVars struct {
	Name          string
	Namespace     string
	StartupConfig string
}

type extraFilesConfigMapTemplateVars struct {
	Name       string
	Namespace  string
	ExtraFiles map[string]string
}

type topologyConfigMapTemplateVars struct {
	NodeName      string
	ConfigMapName string
	FilePath      string
	FileName      string
}

type containerlabTemplateVars struct {
	Name               string
	Namespace          string
	ClabConfig         string
	StartupConfigs     []topologyConfigMapTemplateVars
	ExtraFiles         []topologyConfigMapTemplateVars
	InsecureRegistries []string
}

type renderedContent struct {
	friendlyName string
	fileName     string
	content      []byte
}

func getExtraFilesForNode(
	clabConfig *clabernetesutilcontainerlab.Config,
	nodeName string,
) ([]string, error) {
	var paths []string

	nodeConfig, ok := clabConfig.Topology.Nodes[nodeName]
	if !ok {
		return nil, fmt.Errorf(
			"%w: issue fetching extra files for node '%s', this is a bug",
			ErrClabvert,
			nodeName,
		)
	}

	if nodeConfig.License != "" {
		paths = append(paths, nodeConfig.License)
	} else if clabConfig.Topology.Defaults != nil && clabConfig.Topology.Defaults.License != "" {
		paths = append(paths, clabConfig.Topology.Defaults.License)
	}

	// if there are other files we need to try to load we can just put them here

	return paths, nil
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

	err = os.MkdirAll(c.outputDirectory, clabernetesconstants.PermissionsEveryoneRead)
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
		fullyQualifiedStartupConfigPath := fmt.Sprintf(
			"%s/%s",
			c.topologyPathParent,
			path,
		)

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

func (c *Clabverter) handleStartupConfigs() error {
	c.logger.Info("handling containerlab topology startup config(s) if present...")

	startupConfigs := map[string][]byte{}

	for nodeName, nodeData := range c.clabConfig.Topology.Nodes {
		if nodeData.StartupConfig == "" {
			c.logger.Debugf("node '%s' does not have a startup-config, skipping....", nodeName)

			continue
		}

		c.logger.Debugf("loading node '%s' startup-config...", nodeName)

		startupConfigContents, err := c.resolveContentAtPath(nodeData.StartupConfig)
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
			c.logger.Criticalf(
				"failed loading startup-config configmap template from assets: %s",
				err,
			)

			return err
		}

		configMapName := fmt.Sprintf("%s-%s-startup-config", c.clabConfig.Name, nodeName)

		var rendered bytes.Buffer

		err = t.Execute(
			&rendered,
			startupConfigConfigMapTemplateVars{
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

		fileName := fmt.Sprintf("%s/%s-startup-config.yaml", c.outputDirectory, nodeName)

		c.renderedFiles = append(
			c.renderedFiles,
			renderedContent{
				friendlyName: fmt.Sprintf("%s-statup-config", nodeName),
				fileName:     fileName,
				content:      rendered.Bytes(),
			},
		)

		c.startupConfigConfigMaps[nodeName] = topologyConfigMapTemplateVars{
			NodeName:      nodeName,
			ConfigMapName: configMapName,
			FilePath:      c.clabConfig.Topology.Nodes[nodeName].StartupConfig,
		}
	}

	c.logger.Debug("handling startup configs complete")

	return nil
}

func (c *Clabverter) handleExtraFiles() error {
	c.logger.Info("handling containerlab extra file(s) if present...")

	extraFiles := make(map[string]map[string][]byte)

	for nodeName := range c.clabConfig.Topology.Nodes {
		extraFilePaths, err := getExtraFilesForNode(c.clabConfig, nodeName)
		if err != nil {
			c.logger.Criticalf(
				"failed determining extra file paths for node '%s', error: %s",
				nodeName,
				err,
			)

			return err
		}

		if len(extraFilePaths) == 0 {
			continue
		}

		extraFiles[nodeName] = make(map[string][]byte)

		for _, extraFilePath := range extraFilePaths {
			c.logger.Debugf("loading node '%s' extra file '%s'...", nodeName, extraFilePath)

			var extraFileContent []byte

			extraFileContent, err = c.resolveContentAtPath(extraFilePath)
			if err != nil {
				c.logger.Criticalf(
					"failed loading extra file '%s' contents for node '%s', error: %s",
					extraFilePath,
					nodeName,
					err,
				)

				return err
			}

			extraFiles[nodeName][strings.ReplaceAll(extraFilePath, "/", "_")] = extraFileContent
		}
	}

	c.logger.Info("rendering clabernetes extra file(s) outputs...")

	// render "extra" files
	for nodeName, nodeExtraFiles := range extraFiles {
		t, err := template.ParseFS(Assets, "assets/files-configmap.yaml.template")
		if err != nil {
			c.logger.Criticalf("failed loading files configmap template from assets: %s", err)

			return err
		}

		configMapName := fmt.Sprintf("%s-%s-files", c.clabConfig.Name, nodeName)

		templateVars := extraFilesConfigMapTemplateVars{
			Name:       configMapName,
			Namespace:  c.destinationNamespace,
			ExtraFiles: make(map[string]string),
		}

		c.extraFilesConfigMaps[nodeName] = make([]topologyConfigMapTemplateVars, 0)

		for extraFileName, extraFileContent := range nodeExtraFiles {
			templateVars.ExtraFiles[extraFileName] = "\n" + clabernetesutil.Indent(
				string(extraFileContent),
				specIndentSpaces,
			)

			c.extraFilesConfigMaps[nodeName] = append(
				c.extraFilesConfigMaps[nodeName],
				topologyConfigMapTemplateVars{
					NodeName:      nodeName,
					ConfigMapName: configMapName,
					FilePath:      extraFileName,
					FileName:      extraFileName,
				},
			)
		}

		var rendered bytes.Buffer

		err = t.Execute(&rendered, templateVars)
		if err != nil {
			c.logger.Criticalf("failed executing configmap template: %s", err)

			return err
		}

		fileName := fmt.Sprintf("%s/%s-extra-files.yaml", c.outputDirectory, nodeName)

		c.renderedFiles = append(
			c.renderedFiles,
			renderedContent{
				friendlyName: fmt.Sprintf("%s-statup-config", nodeName),
				fileName:     fileName,
				content:      rendered.Bytes(),
			},
		)
	}

	c.logger.Debug("handling extra file(s) complete")

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
