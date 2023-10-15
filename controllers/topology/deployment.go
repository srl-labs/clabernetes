package topology

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	clabernetesconfig "github.com/srl-labs/clabernetes/config"

	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	k8sappsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) resolveDeployments(
	ctx context.Context,
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) (*clabernetescontrollers.ResolvedDeployments, error) {
	ownedDeployments := &k8sappsv1.DeploymentList{}

	err := r.Client.List(
		ctx,
		ownedDeployments,
		ctrlruntimeclient.InNamespace(obj.GetNamespace()),
		ctrlruntimeclient.MatchingLabels{
			clabernetesconstants.LabelTopologyOwner: obj.GetName(),
		},
	)
	if err != nil {
		r.Log.Criticalf("failed fetching owned deployments, error: '%s'", err)

		return nil, err
	}

	deployments := &clabernetescontrollers.ResolvedDeployments{
		Current: map[string]*k8sappsv1.Deployment{},
	}

	for i := range ownedDeployments.Items {
		labels := ownedDeployments.Items[i].Labels

		if labels == nil {
			return nil, fmt.Errorf(
				"%w: labels are nil, but we expect to see topology owner label here",
				claberneteserrors.ErrInvalidData,
			)
		}

		nodeName, ok := labels[clabernetesconstants.LabelTopologyNode]
		if !ok || nodeName == "" {
			return nil, fmt.Errorf(
				"%w: topology node label is missing or empty",
				claberneteserrors.ErrInvalidData,
			)
		}

		deployments.Current[nodeName] = &ownedDeployments.Items[i]
	}

	allNodes := make([]string, len(clabernetesConfigs))

	var nodeIdx int

	for nodeName := range clabernetesConfigs {
		allNodes[nodeIdx] = nodeName

		nodeIdx++
	}

	deployments.Missing = clabernetesutil.StringSliceDifference(
		deployments.CurrentDeploymentNames(),
		allNodes,
	)

	r.Log.Debugf(
		"deployments are missing for the following nodes: %s",
		deployments.Missing,
	)

	extraEndpointDeployments := clabernetesutil.StringSliceDifference(
		allNodes,
		deployments.CurrentDeploymentNames(),
	)

	r.Log.Debugf(
		"extraneous deployments exist for following nodes: %s",
		extraEndpointDeployments,
	)

	deployments.Extra = make([]*k8sappsv1.Deployment, len(extraEndpointDeployments))

	for idx, endpoint := range extraEndpointDeployments {
		deployments.Extra[idx] = deployments.Current[endpoint]
	}

	return deployments, nil
}

func (r *Reconciler) pruneDeployments(
	ctx context.Context,
	deployments *clabernetescontrollers.ResolvedDeployments,
) error {
	r.Log.Info("pruning extraneous deployments")

	for _, extraDeployment := range deployments.Extra {
		r.Log.Debugf(
			"removing deployment '%s/%s'",
			extraDeployment.Namespace,
			extraDeployment.Name,
		)

		err := r.Client.Delete(ctx, extraDeployment)
		if err != nil {
			r.Log.Criticalf(
				"failed removing deployment '%s/%s' error: %s",
				extraDeployment.Namespace,
				extraDeployment.Name,
				err,
			)

			return err
		}
	}

	return nil
}

func (r *Reconciler) enforceDeployments(
	ctx context.Context,
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	deployments *clabernetescontrollers.ResolvedDeployments,
) error {
	// handle missing deployments
	r.Log.Info("creating missing deployments")

	for _, nodeName := range deployments.Missing {
		deployment := renderDeployment(
			obj,
			nodeName,
		)

		err := ctrlruntimeutil.SetOwnerReference(obj, deployment, r.Client.Scheme())
		if err != nil {
			return err
		}

		r.Log.Debugf(
			"creating deployment '%s/%s'",
			deployment.Namespace,
			deployment.Name,
		)

		err = r.Client.Create(ctx, deployment)
		if err != nil {
			r.Log.Criticalf(
				"failed creating deployment '%s/%s' error: %s",
				deployment.Namespace,
				deployment.Name,
				err,
			)

			return err
		}
	}

	// compare and update existing deployments if we need to
	r.Log.Info("enforcing desired state on existing deployments")

	for nodeName, deployment := range deployments.Current {
		r.Log.Debugf(
			"comparing existing deployment '%s/%s' to desired state",
			deployment.Namespace,
			deployment.Name,
		)

		expectedDeployment := renderDeployment(
			obj,
			nodeName,
		)

		err := ctrlruntimeutil.SetOwnerReference(obj, expectedDeployment, r.Client.Scheme())
		if err != nil {
			return err
		}

		if !deploymentConforms(deployment, expectedDeployment, obj.GetUID()) {
			r.Log.Debugf(
				"comparing existing deployment '%s/%s' spec does not conform to desired state, "+
					"updating",
				deployment.Namespace,
				deployment.Name,
			)

			err = r.Client.Update(ctx, expectedDeployment)
			if err != nil {
				r.Log.Criticalf(
					"failed updating deployment '%s/%s' error: %s",
					expectedDeployment.Namespace,
					expectedDeployment.Name,
					err,
				)

				return err
			}
		}
	}

	return nil
}

func (r *Reconciler) restartDeploymentForNode(
	ctx context.Context,
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	nodeName string,
) error {
	deploymentName := fmt.Sprintf("%s-%s", obj.GetName(), nodeName)

	nodeDeployment := &k8sappsv1.Deployment{}

	err := r.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      deploymentName,
		},
		nodeDeployment,
	)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			r.Log.Warnf(
				"could not find deployment '%s', cannot restart after config change,"+
					" this should not happen",
				deploymentName,
			)

			return nil
		}

		return err
	}

	if nodeDeployment.Spec.Template.ObjectMeta.Annotations == nil {
		nodeDeployment.Spec.Template.ObjectMeta.Annotations = map[string]string{}
	}

	now := time.Now().Format(time.RFC3339)

	nodeDeployment.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"] = now

	return r.Client.Update(ctx, nodeDeployment)
}

func renderDeployment(
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	nodeName string,
) *k8sappsv1.Deployment {
	configManager := clabernetesconfig.GetManager()
	globalAnnotations, globalLabels := configManager.GetAllMetadata()

	name := obj.GetName()

	deploymentName := fmt.Sprintf("%s-%s", name, nodeName)
	configVolumeName := fmt.Sprintf("%s-config", name)

	// match labels are immutable and dont matter if they have the users provided "global" labels,
	// so make those first then copy those into "normal" labels and add the other stuff
	matchLabels := map[string]string{
		clabernetesconstants.LabelApp:           clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:          deploymentName,
		clabernetesconstants.LabelTopologyOwner: name,
		clabernetesconstants.LabelTopologyNode:  nodeName,
	}

	labels := make(map[string]string)

	for k, v := range matchLabels {
		labels[k] = v
	}

	for k, v := range globalLabels {
		labels[k] = v
	}

	deployment := &k8sappsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentName,
			Namespace:   obj.GetNamespace(),
			Annotations: globalAnnotations,
			Labels:      labels,
		},
		Spec: k8sappsv1.DeploymentSpec{
			Replicas:             clabernetesutil.Int32ToPointer(1),
			RevisionHistoryLimit: clabernetesutil.Int32ToPointer(0),
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: globalAnnotations,
					Labels:      labels,
				},
				Spec: k8scorev1.PodSpec{
					Containers: []k8scorev1.Container{
						{
							Name:       nodeName,
							WorkingDir: "/clabernetes",
							Image: clabernetesutil.GetEnvStrOrDefault(
								clabernetesconstants.LauncherImageEnv,
								clabernetesconstants.LauncherDefaultImage,
							),
							Command: []string{"/clabernetes/manager", "launch"},
							Ports: []k8scorev1.ContainerPort{
								{
									Name:          "vxlan",
									ContainerPort: clabernetesconstants.VXLANServicePort,
									Protocol:      "UDP",
								},
							},
							VolumeMounts: []k8scorev1.VolumeMount{
								{
									Name:      configVolumeName,
									ReadOnly:  true,
									MountPath: "/clabernetes/topo.clab.yaml",
									SubPath:   nodeName,
								},
								{
									Name:      configVolumeName,
									ReadOnly:  true,
									MountPath: "/clabernetes/tunnels.yaml",
									SubPath:   fmt.Sprintf("%s-tunnels", nodeName),
								},
							},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: "File",
							ImagePullPolicy: k8scorev1.PullPolicy(
								clabernetesutil.GetEnvStrOrDefault(
									clabernetesconstants.LauncherPullPolicyEnv,
									"IfNotPresent",
								),
							),
							SecurityContext: &k8scorev1.SecurityContext{
								// obviously we need privileged for dind setup
								Privileged: clabernetesutil.BoolToPointer(true),
								RunAsUser:  clabernetesutil.Int64ToPointer(0),
							},
							Env: []k8scorev1.EnvVar{
								{
									Name: clabernetesconstants.LauncherLoggerLevelEnv,
									Value: clabernetesutil.GetEnvStrOrDefault(
										clabernetesconstants.LauncherLoggerLevelEnv,
										clabernetesconstants.Info,
									),
								},
							},
						},
					},
					RestartPolicy:      "Always",
					ServiceAccountName: "default",
					Volumes: []k8scorev1.Volume{
						{
							Name: configVolumeName,
							VolumeSource: k8scorev1.VolumeSource{
								ConfigMap: &k8scorev1.ConfigMapVolumeSource{
									LocalObjectReference: k8scorev1.LocalObjectReference{
										Name: name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if obj.GetTopologyCommonSpec().ContainerlabDebug {
		deployment.Spec.Template.Spec.Containers[0].Env = append(
			deployment.Spec.Template.Spec.Containers[0].Env,
			k8scorev1.EnvVar{
				Name:  clabernetesconstants.LauncherContainerlabDebug,
				Value: clabernetesconstants.True,
			},
		)
	}

	deployment = renderDeploymentAddFilesFromConfigMaps(nodeName, obj, deployment)

	deployment = renderDeploymentAddInsecureRegistries(obj, deployment)

	return deployment
}

func volumeAlreadyMounted(volumeName string, existingVolumes []k8scorev1.Volume) bool {
	for idx := range existingVolumes {
		if volumeName == existingVolumes[idx].Name {
			return true
		}
	}

	return false
}

func renderDeploymentAddFilesFromConfigMaps(
	nodeName string,
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	deployment *k8sappsv1.Deployment,
) *k8sappsv1.Deployment {
	podVolumes := make([]clabernetesapistopologyv1alpha1.FileFromConfigMap, 0)

	for _, fileFromConfigMap := range obj.GetTopologyCommonSpec().FilesFromConfigMap {
		if fileFromConfigMap.NodeName != nodeName {
			continue
		}

		podVolumes = append(podVolumes, fileFromConfigMap)
	}

	for _, podVolume := range podVolumes {
		if !volumeAlreadyMounted(podVolume.ConfigMapName, deployment.Spec.Template.Spec.Volumes) {
			deployment.Spec.Template.Spec.Volumes = append(
				deployment.Spec.Template.Spec.Volumes,
				k8scorev1.Volume{
					Name: podVolume.ConfigMapName,
					VolumeSource: k8scorev1.VolumeSource{
						ConfigMap: &k8scorev1.ConfigMapVolumeSource{
							LocalObjectReference: k8scorev1.LocalObjectReference{
								Name: podVolume.ConfigMapName,
							},
						},
					},
				},
			)
		}

		volumeMount := k8scorev1.VolumeMount{
			Name:      podVolume.ConfigMapName,
			ReadOnly:  false,
			MountPath: fmt.Sprintf("/clabernetes/%s", podVolume.FilePath),
			SubPath:   podVolume.ConfigMapPath,
		}

		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
			volumeMount,
		)
	}

	return deployment
}

func renderDeploymentAddInsecureRegistries(
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	deployment *k8sappsv1.Deployment,
) *k8sappsv1.Deployment {
	insecureRegistries := obj.GetTopologyCommonSpec().InsecureRegistries

	if len(insecureRegistries) > 0 {
		deployment.Spec.Template.Spec.Containers[0].Env = append(
			deployment.Spec.Template.Spec.Containers[0].Env,
			k8scorev1.EnvVar{
				Name:  clabernetesconstants.LauncherInsecureRegistries,
				Value: strings.Join(insecureRegistries, ","),
			},
		)
	}

	return deployment
}

func deploymentConforms(
	existingDeployment,
	renderedDeployment *k8sappsv1.Deployment,
	expectedOwnerUID apimachinerytypes.UID,
) bool {
	if !reflect.DeepEqual(existingDeployment.Spec.Replicas, renderedDeployment.Spec.Replicas) {
		return false
	}

	if !reflect.DeepEqual(existingDeployment.Spec.Selector, renderedDeployment.Spec.Selector) {
		return false
	}

	if !clabernetescontrollers.ContainersEqual(
		existingDeployment.Spec.Template.Spec.Containers,
		renderedDeployment.Spec.Template.Spec.Containers,
	) {
		return false
	}

	if !reflect.DeepEqual(
		existingDeployment.Spec.Template.Spec.ServiceAccountName,
		renderedDeployment.Spec.Template.Spec.ServiceAccountName,
	) {
		return false
	}

	if !reflect.DeepEqual(
		existingDeployment.Spec.Template.Spec.RestartPolicy,
		renderedDeployment.Spec.Template.Spec.RestartPolicy,
	) {
		return false
	}

	if !reflect.DeepEqual(
		existingDeployment.Spec.Template.ObjectMeta.Labels,
		renderedDeployment.Spec.Template.ObjectMeta.Labels,
	) {
		return false
	}

	if existingDeployment.ObjectMeta.Labels == nil {
		// obviously our labels don't exist, so we need to enforce that
		return false
	}

	for k, v := range renderedDeployment.ObjectMeta.Labels {
		var expectedLabelExists bool

		for nk, nv := range existingDeployment.ObjectMeta.Labels {
			if k == nk && v == nv {
				expectedLabelExists = true

				break
			}
		}

		if !expectedLabelExists {
			// missing some expected label, and/or value is wrong
			return false
		}
	}

	if len(existingDeployment.ObjectMeta.OwnerReferences) != 1 {
		// we should have only one owner reference, the extractor
		return false
	}

	if existingDeployment.ObjectMeta.OwnerReferences[0].UID != expectedOwnerUID {
		// owner ref uid is not us
		return false
	}

	return true
}

func determineNodesNeedingRestart(
	preReconcileConfigs,
	configs map[string]*clabernetesutilcontainerlab.Config,
) []string {
	var nodesNeedingRestart []string

	for nodeName, nodeConfig := range configs {
		_, nodeExistedBefore := preReconcileConfigs[nodeName]
		if !nodeExistedBefore {
			continue
		}

		if !reflect.DeepEqual(nodeConfig, preReconcileConfigs[nodeName]) {
			nodesNeedingRestart = append(
				nodesNeedingRestart,
				nodeName,
			)
		}
	}

	return nodesNeedingRestart
}
