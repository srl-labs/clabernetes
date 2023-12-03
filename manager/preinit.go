package manager

import (
	clabernetesgeneratedclientset "github.com/srl-labs/clabernetes/generated/clientset"
	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// preInit handles preparation tasks that happen before running either the init or start methods --
// basically this is stuff that always has to happen before we can do anything.
func (c *clabernetes) preInit() {
	c.logger.Info("initializing cluster info")

	var err error

	c.namespace, err = clabernetesutilkubernetes.CurrentNamespace()
	if err != nil {
		c.logger.Criticalf("failed getting current namespace, err: %s", err)

		c.Panic(err.Error())
	}

	c.kubeConfig, err = rest.InClusterConfig()
	if err != nil {
		c.logger.Criticalf("failed getting in cluster kubeconfig, err: %s", err)

		c.Panic(err.Error())
	}

	c.kubeClient, err = kubernetes.NewForConfig(c.kubeConfig)
	if err != nil {
		c.logger.Criticalf("failed creating kube client from in cluster kubeconfig, err: %s", err)

		c.Panic(err.Error())
	}

	c.kubeClabernetesClient, err = clabernetesgeneratedclientset.NewForConfig(c.kubeConfig)
	if err != nil {
		c.logger.Criticalf(
			"failed creating clabernetes kube client from in cluster kubeconfig, err: %s",
			err,
		)

		c.Panic(err.Error())
	}

	c.scheme = apimachineryruntime.NewScheme()

	c.logger.Debug("initializing cluster info complete...")
}
