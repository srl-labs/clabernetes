package containerlab

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	clabernetescontainerlab "gitlab.com/carlmontanari/clabernetes/containerlab"

	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"

	clabernetescontrollers "gitlab.com/carlmontanari/clabernetes/controllers"
	claberneteserrors "gitlab.com/carlmontanari/clabernetes/errors"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"

	clabernetesapistopologyv1alpha1 "gitlab.com/carlmontanari/clabernetes/apis/topology/v1alpha1"
	clabernetesutil "gitlab.com/carlmontanari/clabernetes/util"
	k8sappsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (c *Controller) reconcileDeployments(
	ctx context.Context,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	preReconcileConfigs,
	configs map[string]*clabernetescontainerlab.Config,
) error {
	deployments, err := c.resolveDeployments(ctx, clab, configs)
	if err != nil {
		return err
	}

	err = c.pruneDeployments(ctx, deployments)
	if err != nil {
		return err
	}

	err = c.enforceDeployments(ctx, clab, deployments)
	if err != nil {
		return err
	}

	nodesNeedingRestart := determineNodesNeedingRestart(preReconcileConfigs, configs)
	if len(nodesNeedingRestart) == 0 {
		return nil
	}

	for _, nodeName := range nodesNeedingRestart {
		if !clabernetesutil.StringSliceContains(deployments.Missing, nodeName) {
			// is a new node, don't restart
			continue
		}

		c.BaseController.Log.Infof(
			"restarting the nodes '%s' as configurations have changed",
			nodesNeedingRestart,
		)

		err = c.restartDeploymentForNode(ctx, clab, nodeName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) resolveDeployments(
	ctx context.Context,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	configs map[string]*clabernetescontainerlab.Config,
) (*clabernetescontrollers.ResolvedDeployments, error) {
	ownedDeployments := &k8sappsv1.DeploymentList{}

	err := c.Client.List(
		ctx,
		ownedDeployments,
		ctrlruntimeclient.InNamespace(clab.Namespace),
		ctrlruntimeclient.MatchingLabels{
			clabernetesconstants.LabelTopologyOwner: clab.Name,
		},
	)
	if err != nil {
		c.Log.Criticalf("failed fetching owned deployments, error: '%s'", err)

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

	allNodes := make([]string, len(configs))

	var nodeIdx int

	for nodeName := range configs {
		allNodes[nodeIdx] = nodeName

		nodeIdx++
	}

	deployments.Missing = clabernetesutil.StringSliceDifference(
		deployments.CurrentDeploymentNames(),
		allNodes,
	)

	c.BaseController.Log.Debugf(
		"deployments are missing for the following nodes: %s",
		deployments.Missing,
	)

	extraEndpointDeployments := clabernetesutil.StringSliceDifference(
		allNodes,
		deployments.CurrentDeploymentNames(),
	)

	c.BaseController.Log.Debugf(
		"extraneous deployments exist for following nodes: %s",
		extraEndpointDeployments,
	)

	deployments.Extra = make([]*k8sappsv1.Deployment, len(extraEndpointDeployments))

	for idx, endpoint := range extraEndpointDeployments {
		deployments.Extra[idx] = deployments.Current[endpoint]
	}

	return deployments, nil
}

func (c *Controller) pruneDeployments(
	ctx context.Context,
	deployments *clabernetescontrollers.ResolvedDeployments,
) error {
	c.BaseController.Log.Info("pruning extraneous deployments")

	for _, extraDeployment := range deployments.Extra {
		c.BaseController.Log.Debugf(
			"removing deployment '%s/%s'",
			extraDeployment.Namespace,
			extraDeployment.Name,
		)

		err := c.Client.Delete(ctx, extraDeployment)
		if err != nil {
			c.BaseController.Log.Criticalf(
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

func (c *Controller) enforceDeployments( //nolint:dupl
	ctx context.Context,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	deployments *clabernetescontrollers.ResolvedDeployments,
) error {
	// handle missing deployments
	c.BaseController.Log.Info("creating missing deployments")

	for _, nodeName := range deployments.Missing {
		deployment := renderDeployment(
			clab,
			nodeName,
		)

		err := c.enforceDeploymentOwnerReference(clab, deployment)
		if err != nil {
			return err
		}

		c.BaseController.Log.Debugf(
			"creating deployment '%s/%s'",
			deployment.Namespace,
			deployment.Name,
		)

		err = c.Client.Create(ctx, deployment)
		if err != nil {
			c.BaseController.Log.Criticalf(
				"failed creating deployment '%s/%s' error: %s",
				deployment.Namespace,
				deployment.Name,
				err,
			)

			return err
		}
	}

	// compare and update existing deployments if we need to
	c.BaseController.Log.Info("enforcing desired state on existing deployments")

	for nodeName, deployment := range deployments.Current {
		c.BaseController.Log.Debugf(
			"comparing existing deployment '%s/%s' to desired state",
			deployment.Namespace,
			deployment.Name,
		)

		expectedDeployment := renderDeployment(
			clab,
			nodeName,
		)

		err := c.enforceDeploymentOwnerReference(clab, expectedDeployment)
		if err != nil {
			return err
		}

		if !deploymentConforms(deployment, expectedDeployment, clab.UID) {
			c.BaseController.Log.Debugf(
				"comparing existing deployment '%s/%s' spec does not conform to desired state, "+
					"updating",
				deployment.Namespace,
				deployment.Name,
			)

			err = c.Client.Update(ctx, expectedDeployment)
			if err != nil {
				c.BaseController.Log.Criticalf(
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

func renderDeployment(
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	nodeName string,
) *k8sappsv1.Deployment {
	deploymentName := fmt.Sprintf("%s-%s", clab.Name, nodeName)
	configVolumeName := fmt.Sprintf("%s-config", clab.Name)

	labels := map[string]string{
		clabernetesconstants.LabelApp:           clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:          deploymentName,
		clabernetesconstants.LabelTopologyOwner: clab.Name,
		clabernetesconstants.LabelTopologyNode:  nodeName,
	}

	deployment := &k8sappsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: clab.Namespace,
			Labels:    labels,
		},
		Spec: k8sappsv1.DeploymentSpec{
			Replicas:             clabernetesutil.Int32ToPointer(1),
			RevisionHistoryLimit: clabernetesutil.Int32ToPointer(0),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
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
									ContainerPort: clabernetesconstants.VXLANPort,
									Protocol:      "UDP",
								},
							},
							VolumeMounts: []k8scorev1.VolumeMount{
								{
									Name:      configVolumeName,
									ReadOnly:  true,
									MountPath: "/clabernetes/topo.yaml",
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
										Name: clab.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	deployment = renderDeploymentAddFilesFromConfigMaps(nodeName, clab, deployment)

	deployment = renderDeploymentAddInsecureRegistries(clab, deployment)

	return deployment
}

func renderDeploymentAddFilesFromConfigMaps(
	nodeName string,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	deployment *k8sappsv1.Deployment,
) *k8sappsv1.Deployment {
	podVolumes := make([]clabernetesapistopologyv1alpha1.FileFromConfigMap, 0)

	for _, fileFromConfigMap := range clab.Spec.FilesFromConfigMap {
		if fileFromConfigMap.NodeName != nodeName {
			continue
		}

		podVolumes = append(podVolumes, fileFromConfigMap)
	}

	for _, podVolume := range podVolumes {
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
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	deployment *k8sappsv1.Deployment,
) *k8sappsv1.Deployment {
	if len(clab.Spec.InsecureRegistries) > 0 {
		deployment.Spec.Template.Spec.Containers[0].Env = append(
			deployment.Spec.Template.Spec.Containers[0].Env,
			k8scorev1.EnvVar{
				Name:  clabernetesconstants.LauncherInsecureRegistries,
				Value: strings.Join(clab.Spec.InsecureRegistries, ","),
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

func (c *Controller) enforceDeploymentOwnerReference(
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	deployment *k8sappsv1.Deployment,
) error {
	err := ctrlruntimeutil.SetOwnerReference(clab, deployment, c.BaseController.Client.Scheme())
	if err != nil {
		c.BaseController.Log.Criticalf(
			"failed setting owner reference on deployment '%s/%s' error: %s",
			deployment.Namespace,
			deployment.Name,
			err,
		)

		return err
	}

	return nil
}

func determineNodesNeedingRestart(
	preReconcileConfigs,
	configs map[string]*clabernetescontainerlab.Config,
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

func (c *Controller) restartDeploymentForNode(
	ctx context.Context,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	nodeName string,
) error {
	deploymentName := fmt.Sprintf("%s-%s", clab.Name, nodeName)

	nodeDeployment := &k8sappsv1.Deployment{}

	err := c.BaseController.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Namespace: clab.Namespace,
			Name:      deploymentName,
		},
		nodeDeployment,
	)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			c.BaseController.Log.Warnf(
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

	return c.BaseController.Client.Update(ctx, nodeDeployment)
}
