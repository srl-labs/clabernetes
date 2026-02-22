package clabverter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	sigsyaml "sigs.k8s.io/yaml"
)

const (
	specIndentSpaces           = 4
	specDefinitionIndentSpaces = 10
	maxBytesForConfigMap       = 950_000
)

// StatuslessTopology is the same as a "normal" Topology without the status field since this field
// should not be present in clabverter output.
type StatuslessTopology struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec clabernetesapisv1alpha1.TopologySpec `json:"spec,omitempty"`
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
	imagePullSecrets   []string

	disableExpose bool

	topologyPath       string
	topologyPathParent string
	isRemotePath       bool

	topologySpecFile     string
	topologySpecFilePath string

	githubGroup         string
	githubRepo          string
	githubToken         string
	naming              string
	containerlabVersion string

	rawClabConfig string
	clabConfig    *clabernetesutilcontainerlab.Config

	// mapping of nodeName -> startup-config info for the templating process; this is its own thing
	// because configurations may be huge and configmaps have a 1M char limit, so while keeping them
	// by themselves may not "solve" for ginormous configs, it can certainly give us a little extra
	// breathing room by not having other data in the startup configmap.
	startupConfigConfigMaps map[string]topologyConfigMapTemplateVars

	// all other config files associated to the node(s) -- for example license file(s).
	extraFilesConfigMaps map[string][]topologyConfigMapTemplateVars

	// any files that are too big for configmaps can be mounted as fileFromURL (if we are "remote"
	// topology at least).
	extraFilesFromURL map[string][]topologyFileFromURLTemplateVars

	// filenames -> content of all rendered files we need to either print to stdout or write to disk
	renderedFiles []renderedContent

	// fromSnapshot is an optional Snapshot CR name (or its backing ConfigMap name) to restore
	// from. When set, filesFromConfigMap entries are pre-populated in the rendered Topology
	// manifest for each node that has saved configs in the snapshot.
	fromSnapshot string
}

// MustNewClabverter returns an instance of Clabverter or panics.
func MustNewClabverter(
	topologyFile,
	topologySpecFile,
	outputDirectory,
	destinationNamespace,
	naming,
	containerlabVersion,
	insecureRegistries string,
	imagePullSecrets string,
	disableExpose,
	debug,
	quiet,
	stdout bool,
	fromSnapshot string,
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

	oldClabverterLogger, _ := logManager.GetLogger(clabernetesconstants.Clabverter)
	if oldClabverterLogger != nil {
		logManager.DeleteLogger(clabernetesconstants.Clabverter)
	}

	clabverterLogger := logManager.MustRegisterAndGetLogger(
		clabernetesconstants.Clabverter,
		logLevel,
	)

	// trim insecureRegistries and split into array if not empty
	var insecureRegistriesArr []string
	if strings.TrimSpace(insecureRegistries) != "" {
		insecureRegistriesArr = strings.Split(insecureRegistries, ",")
	}

	// trim imagePullSecrets and split into array if not empty
	var imagePullSecretsArr []string
	if strings.TrimSpace(imagePullSecrets) != "" {
		imagePullSecretsArr = strings.Split(imagePullSecrets, ",")
	}

	supportedNamings := []string{"prefixed", "non-prefixed"}
	if !slices.Contains(supportedNamings, naming) {
		clabverterLogger.Fatalf(
			"naming flag value is not recognized: %s, possible values %q",
			naming,
			supportedNamings,
		)
	}

	githubToken := os.Getenv(clabernetesconstants.GitHubTokenEnv)

	return &Clabverter{
		logger:                  clabverterLogger,
		topologyFile:            topologyFile,
		topologySpecFile:        topologySpecFile,
		githubToken:             githubToken,
		outputDirectory:         outputDirectory,
		stdout:                  stdout,
		disableExpose:           disableExpose,
		destinationNamespace:    destinationNamespace,
		insecureRegistries:      insecureRegistriesArr,
		imagePullSecrets:        imagePullSecretsArr,
		startupConfigConfigMaps: make(map[string]topologyConfigMapTemplateVars),
		extraFilesConfigMaps:    make(map[string][]topologyConfigMapTemplateVars),
		extraFilesFromURL:       make(map[string][]topologyFileFromURLTemplateVars),
		naming:                  naming,
		containerlabVersion:     containerlabVersion,
		renderedFiles:           []renderedContent{},
		fromSnapshot:            fromSnapshot,
	}
}

// Clabvert is the main (only) entrypoint that kicks off the "clabversion" process.
func (c *Clabverter) Clabvert() error {
	c.logger.Info("starting clabversion!")

	if clabernetesutil.IsURL(c.topologyFile) {
		c.isRemotePath = true

		c.githubGroup, c.githubRepo = clabernetesutil.GitHubGroupAndRepoFromURL(c.topologyFile)

		if c.githubGroup == "" || c.githubRepo == "" {
			c.logger.Warn("topology file is remote but could not parse github group/repo")
		}
	}

	var err error

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

	err = c.handleNamespace()
	if err != nil {
		return err
	}

	err = c.handleAssociatedFiles()
	if err != nil {
		return err
	}

	if c.fromSnapshot != "" {
		err = c.handleFromSnapshot()
		if err != nil {
			return err
		}
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

	err = os.MkdirAll(
		c.outputDirectory,
		clabernetesconstants.PermissionsEveryoneReadWriteOwnerExecute,
	)
	if err != nil {
		c.logger.Criticalf("failed ensuring output directory exists, error: %s", err)

		return err
	}

	return nil
}

func (c *Clabverter) resolveContentAtPath(path string) ([]byte, error) {
	var content []byte

	var err error

	if c.isRemotePath {
		w := &bytes.Buffer{}

		err = clabernetesutil.WriteHTTPContentsFromPath(
			context.Background(),
			clabernetesutil.GitHubNormalToRawLink(fmt.Sprintf("%s/%s", c.topologyPathParent, path)),
			w,
			c.getGitHubHeaders(),
		)

		content = w.Bytes()
	} else {
		fullyQualifiedConfigPath := path

		// set the fully qualified path if the source path is not fully qualified
		if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, c.topologyPathParent) {
			// we may have already set this while processing bind mounts, so don't blindly add the
			// parent path unless we need to!
			fullyQualifiedConfigPath = fmt.Sprintf(
				"%s/%s",
				c.topologyPathParent,
				path,
			)
		}

		content, err = os.ReadFile(fullyQualifiedConfigPath) //nolint:gosec
	}

	return content, err
}

func (c *Clabverter) load() error {
	// get fully qualified path of the topo file
	var err error

	c.logger.Info("loading and validating provided containerlab topology file...")

	if c.isRemotePath {
		rawLink := clabernetesutil.GitHubNormalToRawLink(c.topologyFile)

		if rawLink != c.topologyFile {
			c.logger.Info("converted github link to raw style...")

			c.topologyFile = rawLink
		}
	} else {
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
	if c.isRemotePath {
		pathParts := strings.Split(c.topologyFile, "/")

		c.topologyPathParent = strings.Join(pathParts[:len(pathParts)-1], "/")
	} else {
		c.topologyPathParent = filepath.Dir(c.topologyPath)
	}

	if c.topologySpecFile != "" {
		c.topologySpecFilePath, err = filepath.Abs(c.topologySpecFile)
		if err != nil {
			c.logger.Criticalf("failed determining absolute path of values file, error: %s", err)

			return err
		}
	}

	c.logger.Debugf(
		"determined fully qualified topology spec values file path as: %s", c.topologySpecFilePath,
	)

	c.logger.Debug("attempting to load containerlab topology....")

	var rawClabConfigBytes []byte

	if c.isRemotePath {
		w := &bytes.Buffer{}

		err = clabernetesutil.WriteHTTPContentsFromPath(
			context.Background(),
			clabernetesutil.GitHubNormalToRawLink(c.topologyFile),
			w,
			c.getGitHubHeaders(),
		)

		rawClabConfigBytes = w.Bytes()
	} else {
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

	// set the destination namespace to the c9s-<topology name>
	// if it was not explicitly set via the cli
	if c.destinationNamespace == "" {
		c.destinationNamespace = clabernetesutilkubernetes.SafeConcatNameKubernetes(
			"c9s",
			c.clabConfig.Name,
		)
	}

	if len(c.clabConfig.Topology.Nodes) == 0 {
		c.logger.Info("no nodes in topology file, nothing to do...")

		return nil
	}

	c.logger.Debug("loading and validating containerlab topology file complete!")

	return nil
}

// handleNamespace renders the namespace manifest.
func (c *Clabverter) handleNamespace() error {
	t, err := template.ParseFS(Assets, "assets/namespace.yaml.template")
	if err != nil {
		c.logger.Criticalf("failed loading namespace manifest template from assets: %s", err)

		return err
	}

	var rendered bytes.Buffer

	err = t.Execute(
		&rendered,
		struct {
			Name string
		}{
			Name: c.destinationNamespace,
		},
	)
	if err != nil {
		c.logger.Criticalf("failed executing namespace template: %s", err)

		return err
	}

	// prefix w/ "_" to try to ensure its first file that gets applied when k applying a directory
	fileName := fmt.Sprintf("%s/_%s-ns.yaml", c.outputDirectory, c.clabConfig.Name)

	c.renderedFiles = append(
		c.renderedFiles,
		renderedContent{
			friendlyName: "namespace manifest",
			fileName:     fileName,
			content:      rendered.Bytes(),
		},
	)

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

// handleManifest renders the clabernetes Topology CR manifest.
func (c *Clabverter) handleManifest() error {
	t, err := template.ParseFS(Assets, "assets/topology.yaml.template")
	if err != nil {
		c.logger.Criticalf("failed loading containerlab manifest from assets: %s", err)

		return err
	}

	files := map[string][]topologyConfigMapTemplateVars{}

	for nodeName, startupConfig := range c.startupConfigConfigMaps {
		files[nodeName] = append(files[nodeName], startupConfig)
	}

	for nodeName, nodeExtraFiles := range c.extraFilesConfigMaps {
		files[nodeName] = append(files[nodeName], nodeExtraFiles...)
	}

	// sort the files in the filesFromConfigMap section for more sanity and easier testing :p
	for nodeName := range files {
		sort.Slice(files[nodeName], func(i, j int) bool {
			return files[nodeName][i].FileName < files[nodeName][j].FileName
		})
	}

	var rendered bytes.Buffer

	err = t.Execute(
		&rendered,
		containerlabTemplateVars{
			Name:      c.clabConfig.Name,
			Namespace: c.destinationNamespace,
			// pad w/ a newline so the template can look prettier :)
			ClabConfig: "\n" + clabernetesutil.Indent(
				c.rawClabConfig,
				specDefinitionIndentSpaces,
			),
			Files:               files,
			FilesFromURL:        c.extraFilesFromURL,
			InsecureRegistries:  c.insecureRegistries,
			ImagePullSecrets:    c.imagePullSecrets,
			DisableExpose:       c.disableExpose,
			Naming:              c.naming,
			ContainerlabVersion: c.containerlabVersion,
		},
	)
	if err != nil {
		c.logger.Criticalf("failed executing topology template, error: %s", err)

		return err
	}

	finalRendered, err := c.mergeConfigSpecWithRenderedTopology(rendered.Bytes())
	if err != nil {
		c.logger.Criticalf("failed merging spec config with rendered topology, error: %s", err)

		return err
	}

	fileName := fmt.Sprintf("%s/%s.yaml", c.outputDirectory, c.clabConfig.Name)

	c.renderedFiles = append(
		c.renderedFiles,
		renderedContent{
			friendlyName: "clabernetes manifest",
			fileName:     fileName,
			content:      finalRendered,
		},
	)

	return nil
}

func (c *Clabverter) mergeConfigSpecWithRenderedTopology(
	renderedTopologySpecBytes []byte,
) ([]byte, error) {
	if c.topologySpecFilePath == "" {
		return renderedTopologySpecBytes, nil
	}

	content, err := os.ReadFile(c.topologySpecFilePath)
	if err != nil {
		return nil, err
	}

	topologySpecFromTopoSpecsFile := &clabernetesapisv1alpha1.TopologySpec{}

	err = sigsyaml.Unmarshal(content, topologySpecFromTopoSpecsFile)
	if err != nil {
		return nil, err
	}

	topologyFromTopoSpecsFile := &StatuslessTopology{
		Spec: *topologySpecFromTopoSpecsFile,
	}

	topologyFromTopoSpecsFileBytes, err := sigsyaml.Marshal(topologyFromTopoSpecsFile)
	if err != nil {
		return nil, err
	}

	finalTopology := &StatuslessTopology{}

	err = sigsyaml.Unmarshal(topologyFromTopoSpecsFileBytes, finalTopology)
	if err != nil {
		return nil, err
	}

	err = sigsyaml.Unmarshal(renderedTopologySpecBytes, finalTopology)
	if err != nil {
		return nil, err
	}

	finalTopologyBytes, err := sigsyaml.Marshal(finalTopology)
	if err != nil {
		return nil, err
	}

	// add yaml start document chars
	finalTopologyBytes = append([]byte("---\n"), finalTopologyBytes...)

	return finalTopologyBytes, nil
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
				clabernetesconstants.PermissionsEveryoneReadWriteOwnerExecute,
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

// handleFromSnapshot looks up the snapshot ConfigMap and pre-populates startupConfigConfigMaps
// with filesFromConfigMap entries for each node that has saved configs in the snapshot.
func (c *Clabverter) handleFromSnapshot() error {
	c.logger.Infof("loading snapshot %q to pre-populate filesFromConfigMap...", c.fromSnapshot)

	// Build a kubernetes client from the default kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	).ClientConfig()
	if err != nil {
		return fmt.Errorf("failed building kubeconfig for snapshot lookup: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("failed creating kubernetes client for snapshot lookup: %w", err)
	}

	// Determine the namespace to look up the ConfigMap in
	namespace := c.destinationNamespace
	if namespace == "" {
		namespace = "default"
	}

	// Look up the ConfigMap (the snapshot ConfigMap has the same name as the Snapshot CR)
	configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(
		context.Background(),
		c.fromSnapshot,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf(
			"failed fetching snapshot ConfigMap %q in namespace %q: %w",
			c.fromSnapshot,
			namespace,
			err,
		)
	}

	c.logger.Infof(
		"found snapshot ConfigMap %q with %d keys",
		c.fromSnapshot,
		len(configMap.Data),
	)

	// Group keys by node name (keys are in format "<nodeName>/<fileName>")
	nodeFiles := groupConfigMapKeysByNode(configMap)

	// For each node in the topology that has saved configs, add filesFromConfigMap entries
	for nodeName, fileKeys := range nodeFiles {
		if _, ok := c.clabConfig.Topology.Nodes[nodeName]; !ok {
			c.logger.Debugf(
				"snapshot has data for node %q but it is not in the current topology, skipping",
				nodeName,
			)

			continue
		}

		for _, key := range fileKeys {
			// Determine the mount path: for startup-config files, use the standard path
			// For other files, derive from the key name
			fileName := key[strings.LastIndex(key, "/")+1:]

			// skip save-output files, they are informational only
			if fileName == "save-output" {
				continue
			}

			mountPath := fmt.Sprintf("/clabernetes/%s", fileName)

			// If there's already a startup config for this node from the topology file,
			// use its path; otherwise use a generic path
			if existingEntry, exists := c.startupConfigConfigMaps[nodeName]; exists {
				mountPath = existingEntry.FilePath
			}

			c.logger.Debugf(
				"adding snapshot filesFromConfigMap entry for node %q: key=%q mountPath=%q",
				nodeName,
				key,
				mountPath,
			)

			// Override or add the entry for this node
			c.startupConfigConfigMaps[nodeName] = topologyConfigMapTemplateVars{
				NodeName:      nodeName,
				ConfigMapName: c.fromSnapshot,
				FilePath:      mountPath,
				FileName:      key,
				FileMode:      clabernetesconstants.FileModeRead,
			}

			// Only use the first non-save-output file per node (the startup config)
			break
		}
	}

	c.logger.Info("snapshot filesFromConfigMap entries populated")

	return nil
}

// groupConfigMapKeysByNode groups ConfigMap data keys by node name.
// Keys are expected to be in format "<nodeName>/<fileName>".
func groupConfigMapKeysByNode(configMap *k8scorev1.ConfigMap) map[string][]string {
	nodeFiles := make(map[string][]string)

	for key := range configMap.Data {
		parts := strings.SplitN(key, "/", splitInTwo)
		if len(parts) != splitInTwo {
			continue
		}

		nodeName := parts[0]
		nodeFiles[nodeName] = append(nodeFiles[nodeName], key)
	}

	return nodeFiles
}

const splitInTwo = 2

func (c *Clabverter) getGitHubHeaders() map[string]string {
	if c.githubToken == "" {
		return nil
	}

	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", c.githubToken),
	}
}
