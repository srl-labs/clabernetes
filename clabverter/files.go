package clabverter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"

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

	// we may split on a trailing ro/rw -- we'll ignore it in clabernetes case since this will be
	// mounted as the pods own configmap anyway and not something we need to care about not getting
	// overwritten locally like in the normal containerlab case
	if len(bindParts) < bindPartsLen {
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

		if len(startupConfigContents) > maxBytesForConfigMap {
			panic("STARTUP CONFIG TOO LARGE, REMOTE STARTUP CONFIG NOT IMPLEMENTED YET")
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
			FileName:      "startup-config",
		}
	}

	c.logger.Debug("handling startup configs complete")

	return nil
}

// TODO move me to files.
func (c *Clabverter) resolveExtraFiles(
	extraFilePaths,
	resolvedExtraFilePaths []sourceDestinationPathPair,
) ([]sourceDestinationPathPair, error) {
	for _, extraFilePath := range extraFilePaths {
		if c.isRemotePath {
			// TODO need to do it like curl the shit below:
			//  https://api.github.com/repos/srl-labs/srl-telemetry-lab/contents/configs/client3
			panic("notimplemented")
		} else {
			fullyQualifiedPath := extraFilePath.sourcePath

			if !strings.HasPrefix(extraFilePath.sourcePath, c.topologyPathParent) {
				// we may have already set this while processing bind mounts, so don't blindly add
				// the parent path unless we need to!
				fullyQualifiedPath = fmt.Sprintf(
					"%s/%s",
					c.topologyPathParent,
					extraFilePath.sourcePath,
				)
			}

			fileInfo, err := os.Stat(fullyQualifiedPath)
			if err != nil {
				return nil, fmt.Errorf(
					"%w: failed stat'ing file or directory, err: %w", ErrClabvert, err,
				)
			}

			if !fileInfo.IsDir() {
				resolvedExtraFilePaths = append(
					resolvedExtraFilePaths,
					sourceDestinationPathPair{
						sourcePath:      fullyQualifiedPath,
						destinationPath: extraFilePath.destinationPath,
					},
				)

				continue
			}

			extraFileSubPaths, err := filepath.Glob(fmt.Sprintf("%s/*", fullyQualifiedPath))
			if err != nil {
				return nil, fmt.Errorf(
					"%w: failed glob'ing directory, err: %w", ErrClabvert, err,
				)
			}

			for _, subExtraFileSubPath := range extraFileSubPaths {
				resolvedExtraFilePaths, err = c.resolveExtraFiles(
					[]sourceDestinationPathPair{
						{
							sourcePath: subExtraFileSubPath,
							destinationPath: strings.TrimPrefix(
								subExtraFileSubPath,
								c.topologyPathParent,
							),
						},
					},
					resolvedExtraFilePaths,
				)
				if err != nil {
					return nil, fmt.Errorf(
						"%w: error recursively loading extra files, err: %w", ErrClabvert, err,
					)
				}
			}
		}
	}

	return resolvedExtraFilePaths, nil
}

// handleExtraFiles deals with parsing/loading/rendering "extra" files for a containerlab topology.
// this means dealing with any non startup-config file basically -- so things like license and any
// bind mounts.
func (c *Clabverter) handleExtraFiles() error {
	c.logger.Info("handling containerlab extra file(s) if present...")

	extraFiles := make(map[string]map[string][]byte)

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

		resolvedExtraFiles, err := c.resolveExtraFiles(extraFilePaths, nil)
		if err != nil {
			c.logger.Criticalf(
				"failed resolving extra file paths for node '%s', error: %s",
				nodeName,
				err,
			)

			return err
		}

		extraFiles[nodeName] = make(map[string][]byte)

		for _, extraFilePath := range resolvedExtraFiles {
			c.logger.Debugf(
				"loading node '%s' extra file '%s' for destination '%s'...",
				nodeName,
				extraFilePath.sourcePath,
				extraFilePath.destinationPath,
			)

			var extraFileContent []byte

			extraFileContent, err = c.resolveContentAtPath(extraFilePath.sourcePath)
			if err != nil {
				c.logger.Criticalf(
					"failed loading extra file '%s' contents for node '%s', error: %s",
					extraFilePath,
					nodeName,
					err,
				)

				return err
			}

			if len(extraFileContent) > maxBytesForConfigMap {
				// file is too big for a configmap

				if c.isRemotePath {
					_, ok := c.extraFilesFromURL[nodeName]
					if !ok {
						c.extraFilesFromURL[nodeName] = make([]topologyFileFromURLTemplateVars, 0)
					}

					c.extraFilesFromURL[nodeName] = append(
						c.extraFilesFromURL[nodeName],
						topologyFileFromURLTemplateVars{
							URL:      extraFilePath.sourcePath,
							FilePath: extraFilePath.destinationPath,
						},
					)
				} else {
					panic("NOT REMOTE PATH BUT FILE IS TOO BIG FOR CONFIGMAP")
				}
			} else {
				extraFiles[nodeName][extraFilePath.destinationPath] = extraFileContent
			}
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

		for extraFilePath, extraFileContent := range nodeExtraFiles {
			safeFileName := clabernetesutilkubernetes.SafeConcatNameKubernetes(
				strings.Split(extraFilePath, "/")...)

			safeFileName = strings.TrimPrefix(safeFileName, "-")

			templateVars.ExtraFiles[safeFileName] = "\n" + clabernetesutil.Indent(
				string(extraFileContent),
				specIndentSpaces,
			)

			c.extraFilesConfigMaps[nodeName] = append(
				c.extraFilesConfigMaps[nodeName],
				topologyConfigMapTemplateVars{
					NodeName:      nodeName,
					ConfigMapName: configMapName,
					FilePath:      extraFilePath,
					FileName:      safeFileName,
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
				friendlyName: fmt.Sprintf("%s extra files", nodeName),
				fileName:     fileName,
				content:      rendered.Bytes(),
			},
		)
	}

	c.logger.Debug("handling extra file(s) complete")

	return nil
}
