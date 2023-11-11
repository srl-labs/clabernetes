package config

import (
	"context"
	"fmt"
	"strings"
	"sync"

	apimachinerywatch "k8s.io/apimachinery/pkg/watch"

	"gopkg.in/yaml.v3"

	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	managerInstance     Manager   //nolint:gochecknoglobals
	managerInstanceOnce sync.Once //nolint:gochecknoglobals
)

// ManagerGetterFunc returns an instance of the config manager.
type ManagerGetterFunc func() Manager

// InitManager initializes the config manager -- it does this once only, its a no-op if the manager
// is already initialized.
func InitManager(ctx context.Context, appName, namespace string, client *kubernetes.Clientset) {
	managerInstanceOnce.Do(func() {
		logManager := claberneteslogging.GetManager()

		logger := logManager.MustRegisterAndGetLogger(
			"config-manager",
			clabernetesutil.GetEnvStrOrDefault(
				clabernetesconstants.ManagerLoggerLevelEnv,
				clabernetesconstants.Info,
			),
		)

		m := &manager{
			ctx:        ctx,
			logger:     logger,
			appName:    appName,
			namespace:  namespace,
			kubeClient: client,
			lock:       &sync.RWMutex{},
			config: &global{
				globalAnnotations: make(map[string]string),
				globalLabels:      make(map[string]string),
				defaultResources: resources{
					Default:            nil,
					ByContainerlabKind: nil,
				},
			},
		}

		managerInstance = m
	})
}

// GetManager returns the config manager -- if the manager has not been initialized it panics.
func GetManager() Manager {
	if managerInstance == nil {
		panic(
			"config manager instance is nil, 'GetManager' should never be called until the " +
				"manager process has been started",
		)
	}

	return managerInstance
}

// Manager is the config manager interface defining the config manager methods.
type Manager interface {
	// Start starts the config manager -- this should be called by the clabernetes manager during
	// the "pre-start" phase -- we want this to happen prior to starting controller-runtime manager
	// things. This method attempts to find the clabernetes config configmap and then watches that
	// object. *If* this configmap cannot be found we will log a WARN error but *will not crash*
	// -- we will simply return default (empty) values when the config manager is queried by the
	// controllers.
	Start() error
	// GetGlobalAnnotations returns a map of the "global" annotations from the config -- these are
	// annotations that should be applied to *all* clabernetes objects.
	GetGlobalAnnotations() map[string]string
	// GetGlobalLabels returns a map of the "global" labels from the config -- these are labels
	// that should be applied to *all* clabernetes objects.
	GetGlobalLabels() map[string]string
	// GetAllMetadata returns the global annotations and global labels.
	GetAllMetadata() (map[string]string, map[string]string)
	// GetResourcesForContainerlabKind returns the desired default resources for a containerlab
	// kind/type combo.
	GetResourcesForContainerlabKind(
		containerlabKind string,
		containerlabType string,
	) *k8scorev1.ResourceRequirements
}

type manager struct {
	ctx        context.Context
	logger     claberneteslogging.Instance
	appName    string
	namespace  string
	kubeClient *kubernetes.Clientset
	lock       *sync.RWMutex
	started    bool
	lastHash   string
	config     *global
}

func (m *manager) load(configMap *k8scorev1.ConfigMap) {
	m.logger.Debug("re-loading config contents...")

	dataBytes, err := yaml.Marshal(configMap.Data)
	if err != nil {
		m.logger.Warnf("failed marshaling config contents to bytes, error: %s", err)

		return
	}

	newHash := clabernetesutil.HashBytes(dataBytes)

	if m.lastHash == newHash {
		m.logger.Debug("config contents hash matches last recorded hash, nothing to do")

		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	newConfig, err := configFromMap(configMap.Data)
	if err != nil {
		m.logger.Warnf(
			"encountered one or more errors parsing global config configmap, errors: %s", err,
		)
	}

	// filter out any "reserved" labels (we dont use annotations for anything so those can be left
	// alone) -- this means anything starting w/ "appname/" i guess
	for k := range newConfig.globalLabels {
		if strings.HasPrefix(k, fmt.Sprintf("%s/", m.appName)) {
			m.logger.Warnf(
				"removing user provided global label '%s' labels starting with '%s/' are reserved",
				k,
				m.appName,
			)

			delete(newConfig.globalLabels, k)
		}
	}

	m.config = newConfig
}

func (m *manager) Start() error {
	if m.started {
		m.logger.Info("attempting to start already started config manager, this is a no-op")

		return nil
	}

	found := true

	configMap, err := m.kubeClient.CoreV1().
		ConfigMaps(m.namespace).
		Get(m.ctx, fmt.Sprintf("%s-config", m.appName), metav1.GetOptions{})
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			m.logger.Warn(
				"did not find clabernetes global config configmap, will continue but no global" +
					" configs will be applied until/unless this configmap shows up!",
			)

			found = false
		} else {
			m.logger.Criticalf("encountered error fetching global config configmap, err: ", err)

			return err
		}
	}

	if found {
		// if we found the configmap we always load it up or at least check if the hash changed and
		// then load it up
		m.load(configMap)
	}

	m.started = true

	m.logger.Debug("starting config watch go routine and running forever or until sigint...")

	go m.watchConfigMap()

	return nil
}

func (m *manager) GetGlobalAnnotations() map[string]string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	// we dont want to pass by ref, so make a new map
	outAnnotations := make(map[string]string)

	for k, v := range m.config.globalAnnotations {
		outAnnotations[k] = v
	}

	return outAnnotations
}

func (m *manager) GetGlobalLabels() map[string]string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	// we dont want to pass by ref, so make a new map
	outLabels := make(map[string]string)

	for k, v := range m.config.globalLabels {
		outLabels[k] = v
	}

	return outLabels
}

func (m *manager) GetAllMetadata() (outAnnotations, outLabels map[string]string) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	outAnnotations = make(map[string]string)

	for k, v := range m.config.globalAnnotations {
		outAnnotations[k] = v
	}

	outLabels = make(map[string]string)

	for k, v := range m.config.globalLabels {
		outLabels[k] = v
	}

	return outAnnotations, outLabels
}

func (m *manager) GetResourcesForContainerlabKind(
	containerlabKind string,
	containerlabType string,
) *k8scorev1.ResourceRequirements {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.resourcesForContainerlabKind(
		containerlabKind,
		containerlabType,
	)
}

func (m *manager) watchConfigMap() {
	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s-config", m.appName),
		Watch:         true,
	}

	watch, err := m.kubeClient.CoreV1().ConfigMaps(m.namespace).Watch(m.ctx, listOptions)
	if err != nil {
		m.logger.Criticalf("failed watching clabernetes config configmap, err: %s", err)
	}

	for event := range watch.ResultChan() {
		switch event.Type { //nolint:exhaustive
		case apimachinerywatch.Added, apimachinerywatch.Modified:
			m.logger.Info("processing global config configmap add or modification event")

			configMap, ok := event.Object.(*k8scorev1.ConfigMap)
			if !ok {
				m.logger.Warn("failed casting event object to configmap, this is probably a bug")

				continue
			}

			m.load(configMap)
		case apimachinerywatch.Deleted:
			m.logger.Warn(
				"global config configmap was *deleted*, will continue with empty config...",
			)

			m.load(&k8scorev1.ConfigMap{})
		}
	}
}
