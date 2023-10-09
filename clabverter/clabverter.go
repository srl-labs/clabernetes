package clabverter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	kyaml "sigs.k8s.io/yaml"

	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
)

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

	// trim insecureRegistries and split into array if not empty
	var insecureRegistriesArr []string
	if len(strings.TrimSpace(insecureRegistries)) > 0 {
		insecureRegistriesArr = strings.Split(insecureRegistries, ",")
	}

	return &Clabverter{
		logger:               clabverterLogger,
		topologyFile:         topologyFile,
		outputDirectory:      outputDirectory,
		stdout:               stdout,
		destinationNamespace: destinationNamespace,
		insecureRegistries:   insecureRegistriesArr,
		configMaps:           []*corev1.ConfigMap{},
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

	// kubernetes ConfigMaps
	configMaps []*corev1.ConfigMap

	ClabKrm *v1alpha1.Containerlab
}

// Clabvert is the main (only) entrypoint that kicks off the "clabversion" process.
func (c *Clabverter) Clabvert() error {
	var err error
	c.logger.Info("starting clabversion!")

	if !c.stdout {
		err = c.ensureOutputDirectory()
		if err != nil {
			return err
		}
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

	c.topologyPathParent, _ = filepath.Split(c.topologyFile)

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
	c.ClabKrm = NewContainerlabKRM(c.clabConfig.Name, c.destinationNamespace, nil)

	// set insecure registries
	c.ClabKrm.Spec.InsecureRegistries = c.insecureRegistries

	// set config
	c.ClabKrm.Spec.Config = c.rawClabConfig

	return nil
}

func (c *Clabverter) output() error {
	var err error
	if err = c.outputConfigMaps(); err != nil {
		c.logger.Criticalf("failed marshalling ConfigMaps: %s", err)
		return err
	}
	if err = c.outputContainerlab(); err != nil {
		c.logger.Criticalf("failed marshalling Containerlab: %s", err)
		return err
	}
	return nil
}

func (c *Clabverter) outputContainerlab() error {

	data, err := kyaml.Marshal(c.ClabKrm)
	if err != nil {
		c.logger.Criticalf(
			"failed marshalling containerlab krm",
			err,
		)

		return err
	}

	if c.stdout {
		os.Stdout.WriteString("---\n")
		_, err := os.Stdout.Write(data)
		if err != nil {
			c.logger.Criticalf(
				"failed writing containerlab krm to stdout: %s", err,
			)

			return err
		}
	} else {
		filename := fmt.Sprintf("%s.yaml", c.clabConfig.Name)
		err := os.WriteFile(
			filename,
			data,
			clabernetesconstants.PermissionsEveryoneRead,
		)
		if err != nil {
			c.logger.Criticalf(
				"failed writing '%s' to output directory: %s",
				filename,
				err,
			)

			return err
		}
	}

	return nil
}

func (c *Clabverter) outputConfigMaps() error {
	for _, configMap := range c.configMaps {
		os.Stdout.WriteString("---\n")
		data, err := kyaml.Marshal(configMap)
		if err != nil {
			c.logger.Criticalf(
				"failed marshalling %s/%s",
				configMap.Kind, configMap.Name,
				err,
			)

			return err
		}

		if c.stdout {
			_, err := os.Stdout.Write(data)
			if err != nil {
				c.logger.Criticalf(
					"failed writing '%s/%s' to stdout: %s", configMap.Kind, configMap.Name, err,
				)

				return err
			}
		} else {
			err := os.WriteFile(
				fmt.Sprintf("%s/%s", configMap.Kind, configMap.Name),
				data,
				clabernetesconstants.PermissionsEveryoneRead,
			)
			if err != nil {
				c.logger.Criticalf(
					"failed writing '%s/%s' to output directory: %s",
					configMap.Kind, configMap.Name,
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
