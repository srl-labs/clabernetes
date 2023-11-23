package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	apimachinerywatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"

	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	claberneteshttptypes "github.com/srl-labs/clabernetes/http/types"
)

const (
	imageRoute = "/image"
	puller     = "puller"
)

func (m *manager) imageHandler(w http.ResponseWriter, r *http.Request) {
	m.logRequest(r)

	imageRequest, err := processImageRequest(r)
	if err != nil {
		msg := fmt.Sprintf("encountered error processing image request, error: %s", err)

		m.logger.Critical(msg)

		w.WriteHeader(http.StatusInternalServerError)

		_, err = w.Write([]byte(msg))
		if err != nil {
			m.logger.Criticalf(
				"failed writing error message to image request response, error: %s",
				err,
			)
		}

		return
	}

	m.logger.Debugf(
		"received image pull request from pod '%s/%s' in topology %q on node %q,"+
			" requesting image %q",
		imageRequest.TopologyNamespace,
		imageRequest.RequestingPodName,
		imageRequest.TopologyName,
		imageRequest.KubernetesNodeName,
		imageRequest.RequestedImageName,
	)

	// we run the spawning in the background to not block the launcher or http server. it will
	// always try to clean up so it should be ok... fingers crossed?!
	go func() {
		err = spawnImagePullerPod(m.ctx, m.client, m.kubeClient, imageRequest)
		if err != nil {
			m.logger.Criticalf(
				"handling image pull pod for requesting pod '%s/%s' in topology %q on node %q,"+
					" requesting image %q failed, err: %s",
				imageRequest.TopologyNamespace,
				imageRequest.RequestingPodName,
				imageRequest.TopologyName,
				imageRequest.KubernetesNodeName,
				imageRequest.RequestedImageName,
				err,
			)
		} else {
			m.logger.Debugf(
				"handling image pull pod for requesting pod '%s/%s' in topology %q on node %q,"+
					" requesting image %q completed",
				imageRequest.TopologyNamespace,
				imageRequest.RequestingPodName,
				imageRequest.TopologyName,
				imageRequest.KubernetesNodeName,
				imageRequest.RequestedImageName,
			)
		}
	}()

	w.WriteHeader(http.StatusOK)
}

func processImageRequest(
	r *http.Request,
) (*claberneteshttptypes.ImageRequest, error) {
	imageRequest := &claberneteshttptypes.ImageRequest{}

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(reqBody, imageRequest)
	if err != nil {
		return nil, err
	}

	return imageRequest, nil
}

func spawnImagePullerPod(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	kubeClient *kubernetes.Clientset,
	imageRequest *claberneteshttptypes.ImageRequest,
) (reterr error) {
	globalAnnotations, globalLabels := clabernetesconfig.GetManager().GetAllMetadata()

	imageHash := clabernetesutil.HashBytes([]byte(imageRequest.RequestedImageName))

	podName := clabernetesutilkubernetes.SafeConcatNameKubernetes(
		clabernetesconstants.Clabernetes,
		puller,
		imageRequest.KubernetesNodeName,
		imageHash,
	)

	selectorLabels := map[string]string{
		clabernetesconstants.LabelApp: clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName: fmt.Sprintf(
			"%s-%s",
			clabernetesconstants.Clabernetes,
			puller,
		),
		clabernetesconstants.LabelTopologyOwner:    imageRequest.TopologyName,
		clabernetesconstants.LabelTopologyNode:     imageRequest.TopologyNodeName,
		clabernetesconstants.LabelPullerNodeTarget: imageRequest.TopologyNodeName,
		clabernetesconstants.LabelPullerImageHash:  imageHash[:13],
	}

	labels := make(map[string]string)

	for k, v := range selectorLabels {
		labels[k] = v
	}

	for k, v := range globalLabels {
		labels[k] = v
	}

	annotations := map[string]string{
		// image string wont be valid for a label, so we'll put it here
		"clabernetes/pullerRequestedImage": imageRequest.RequestedImageName,
	}

	for k, v := range globalAnnotations {
		annotations[k] = v
	}

	requestedPullSecrets := make(
		[]k8scorev1.LocalObjectReference,
		len(imageRequest.ConfiguredPullSecrets),
	)

	for idx, pullSecret := range imageRequest.ConfiguredPullSecrets {
		requestedPullSecrets[idx] = k8scorev1.LocalObjectReference{Name: pullSecret}
	}

	createCtx, cancel := context.WithTimeout(
		ctx,
		clabernetesconstants.DefaultClientOperationTimeout,
	)

	pullerPod := &k8scorev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        podName,
			Namespace:   imageRequest.TopologyNamespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: k8scorev1.PodSpec{
			Containers: []k8scorev1.Container{
				{
					Name:  "puller",
					Image: imageRequest.RequestedImageName,
					// we don't care if it runs, only care if we can pull the image...
					Command: []string{
						"exit",
						"0",
					},
					ImagePullPolicy: "IfNotPresent",
				},
			},
			NodeName:         imageRequest.KubernetesNodeName,
			ImagePullSecrets: requestedPullSecrets,
		},
	}

	defer func() {
		err := client.Delete(ctx, pullerPod)
		if !apimachineryerrors.IsNotFound(err) {
			reterr = err
		}
	}()

	err := client.Create(
		createCtx,
		pullerPod,
	)

	cancel()

	if err != nil {
		return err
	}

	watchCtx, cancel := context.WithTimeout(ctx, clabernetesconstants.PullerPodTimeout)
	defer cancel()

	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", podName),
		Watch:         true,
	}

	watch, reterr := kubeClient.CoreV1().Pods(pullerPod.Namespace).Watch(watchCtx, listOptions)
	if reterr != nil {
		return err
	}

	for event := range watch.ResultChan() {
		switch event.Type { //nolint:exhaustive
		case apimachinerywatch.Added, apimachinerywatch.Modified:
			pod, ok := event.Object.(*k8scorev1.Pod)
			if !ok {
				panic("this is a bug in the puller pod watch, this should not happen")
			}

			switch pod.Status.Phase { //nolint:exhaustive
			case k8scorev1.PodPending:
				continue
			case k8scorev1.PodRunning, k8scorev1.PodSucceeded, k8scorev1.PodFailed:
				// its running/succeeded/failed any of which means the image has been pulled
				watch.Stop()
			}
		}
	}

	return reterr
}
