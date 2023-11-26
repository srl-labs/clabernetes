package manager

import (
	"fmt"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"

	clabernetesutil "github.com/srl-labs/clabernetes/util"
	"k8s.io/apimachinery/pkg/labels"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimecache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimemetricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func mustNewManager(scheme *apimachineryruntime.Scheme, appName string) ctrlruntime.Manager {
	mgr, err := ctrlruntime.NewManager(
		ctrlruntime.GetConfigOrDie(),
		ctrlruntime.Options{
			Logger: klog.NewKlogr(),
			Scheme: scheme,
			Metrics: ctrlruntimemetricsserver.Options{
				BindAddress: "0",
			},
			LeaderElection: false,
			NewCache: func(
				config *rest.Config,
				opts ctrlruntimecache.Options,
			) (ctrlruntimecache.Cache, error) {
				opts.DefaultLabelSelector = labels.SelectorFromSet(
					labels.Set{
						// only cache objects with the "clabernetes/app" label, why would we care
						// about anything else (for now -- and we can override it with opts.ByObject
						// anyway?! and... who the hell calls their app "clabernetes" so this should
						// really limit the cache nicely :)
						"clabernetes/app": appName,
					},
				)

				opts.ByObject = map[ctrlruntimeclient.Object]ctrlruntimecache.ByObject{
					// obviously we need to cache all "our" topology objects, so do that
					&clabernetesapisv1alpha1.Topology{}: {
						Namespaces: map[string]ctrlruntimecache.Config{
							ctrlruntimecache.AllNamespaces: {
								LabelSelector: labels.Everything(),
							},
						},
					},
				}

				return ctrlruntimecache.New(config, opts)
			},
		},
	)
	if err != nil {
		clabernetesutil.Panic(fmt.Sprintf("unable to start manager, error: %s", err))
	}

	return mgr
}
