package clabverter

import (
	"fmt"
	"strings"

	"github.com/srl-labs/clabernetes/constants"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
)

const (
	bindSeperator   = ":"
	bindPartsLen    = 2
	bindClabNodeDir = "__clabNodeDir__"
	bindClabDir     = "__clabDir__"
)

func parseBindString(bind, nodeName, topologyPathParent string) (sourceDestinationPathPair, error) {
	parsedBind := sourceDestinationPathPair{}

	bindParts := strings.Split(bind, bindSeperator)

	if len(bindParts) != bindPartsLen {
		return parsedBind, fmt.Errorf("%w: bind string %q could not be parsed", ErrClabvert, bind)
	}

	parsedBind.sourcePath = bindParts[0]
	parsedBind.destinationPath = bindParts[1]

	// handle special bind path vars
	parsedBind.sourcePath = strings.Replace(
		parsedBind.sourcePath,
		bindClabNodeDir,
		fmt.Sprintf("%s/%s", topologyPathParent, nodeName),
		1,
	)
	parsedBind.sourcePath = strings.Replace(
		parsedBind.sourcePath, bindClabDir, topologyPathParent, 1,
	)

	return parsedBind, nil
}

func getExtraFilesForNode(
	clabConfig *clabernetesutilcontainerlab.Config,
	nodeName,
	topologyPathParent string,
) ([]sourceDestinationPathPair, error) {
	nodeConfig, ok := clabConfig.Topology.Nodes[nodeName]
	if !ok {
		return nil, fmt.Errorf(
			"%w: issue fetching extra files for node %q, this is a bug",
			ErrClabvert,
			nodeName,
		)
	}

	defaults := clabConfig.Topology.Defaults

	paths := make([]sourceDestinationPathPair, 0)

	if nodeConfig.License != "" {
		paths = append(
			paths,
			sourceDestinationPathPair{
				sourcePath:      nodeConfig.License,
				destinationPath: nodeConfig.License,
			},
		)
	} else if defaults != nil && defaults.License != "" {
		paths = append(
			paths,
			sourceDestinationPathPair{
				sourcePath:      defaults.License,
				destinationPath: defaults.License,
			},
		)
	}

	var allBinds []string

	if defaults != nil {
		// always put default binds in (if they exist)
		allBinds = append(allBinds, defaults.Binds...)
	}

	allBinds = append(allBinds, nodeConfig.Binds...)

	for _, bind := range allBinds {
		parsedBind, err := parseBindString(bind, nodeName, topologyPathParent)
		if err != nil {
			return nil, err
		}

		paths = append(
			paths,
			parsedBind,
		)
	}

	return paths, nil
}

// handleStartupConfigs parses/loads/renders the startup-config(s) for a topology -- this is its own
// thing in part because the startup config configmap is its own thing, we do this because of size
// imitations of configmaps, so best keep startups by themselves in case they are huge.
func (c *Clabverter) handleStartupConfigs() error {

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

		// generate config map name
		configMapName := fmt.Sprintf("%s-%s-startup-config", c.clabConfig.Name, nodeName)

		// Generate labels
		configMapLabels := map[string]string{
			constants.LabelKRMKind:      constants.LabelKRMKindStartupConfig,
			constants.LabelTopologyNode: nodeName,
		}

		// create config map
		configMap := NewConfigMapKRM(configMapName, c.destinationNamespace, configMapLabels)

		// set the configMap Startup-Config data
		configMap.Data["startup-config"] = string(startupConfigContents)

		// append to ConfigMaps
		c.configMaps = append(c.configMaps, configMap)
	}

	c.logger.Debug("handling startup configs complete")

	return nil
}

// handleExtraFiles deals with parsing/loading/rendering "extra" files for a containerlab topology.
// this means dealing with any non startup-config file basically -- so things like license and any
// bind mounts.
func (c *Clabverter) handleExtraFiles() error {
	c.logger.Info("handling containerlab extra file(s) if present...")

	for nodeName := range c.clabConfig.Topology.Nodes {
		extraFilePaths, err := getExtraFilesForNode(c.clabConfig, nodeName, c.topologyPathParent)
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

		configMapName := fmt.Sprintf("%s-%s-files", c.clabConfig.Name, nodeName)
		// Generate labels
		configMapLabels := map[string]string{
			constants.LabelKRMKind:      constants.LabelKRMKindExtraFiles,
			constants.LabelTopologyNode: nodeName,
		}

		configMap := NewConfigMapKRM(configMapName, c.destinationNamespace, configMapLabels)

		// add config map to clabverter configmaps
		c.configMaps = append(c.configMaps, configMap)

		// iterate over extra files adding them to the ConfigMap
		for _, extraFilePath := range extraFilePaths {
			c.logger.Debugf("loading node '%s' extra file '%s'...", nodeName, extraFilePath)

			// resolve filepath and read content
			extraFileContent, err := c.resolveContentAtPath(extraFilePath.sourcePath)
			if err != nil {
				c.logger.Criticalf(
					"failed loading extra file '%s' contents for node '%s', error: %s",
					extraFilePath,
					nodeName,
					err,
				)

				return err
			}
			// convert filename to save filename
			safeFileName := clabernetesutil.SafeConcatNameKubernetes(
				strings.Split(extraFilePath.destinationPath, "/")...)

			safeFileName = strings.TrimPrefix(safeFileName, "-")

			// finally add date to ConfigMap
			configMap.Data[safeFileName] = string(extraFileContent)
		}
	}
	c.logger.Debug("handling extra file(s) complete")
	return nil
}
