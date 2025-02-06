/*
  Copyright The Kubernetes Authors.

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	context "context"

	apisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	scheme "github.com/srl-labs/clabernetes/generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// ImageRequestsGetter has a method to return a ImageRequestInterface.
// A group's client should implement this interface.
type ImageRequestsGetter interface {
	ImageRequests(namespace string) ImageRequestInterface
}

// ImageRequestInterface has methods to work with ImageRequest resources.
type ImageRequestInterface interface {
	Create(
		ctx context.Context,
		imageRequest *apisv1alpha1.ImageRequest,
		opts v1.CreateOptions,
	) (*apisv1alpha1.ImageRequest, error)
	Update(
		ctx context.Context,
		imageRequest *apisv1alpha1.ImageRequest,
		opts v1.UpdateOptions,
	) (*apisv1alpha1.ImageRequest, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(
		ctx context.Context,
		imageRequest *apisv1alpha1.ImageRequest,
		opts v1.UpdateOptions,
	) (*apisv1alpha1.ImageRequest, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*apisv1alpha1.ImageRequest, error)
	List(ctx context.Context, opts v1.ListOptions) (*apisv1alpha1.ImageRequestList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(
		ctx context.Context,
		name string,
		pt types.PatchType,
		data []byte,
		opts v1.PatchOptions,
		subresources ...string,
	) (result *apisv1alpha1.ImageRequest, err error)
	ImageRequestExpansion
}

// imageRequests implements ImageRequestInterface
type imageRequests struct {
	*gentype.ClientWithList[*apisv1alpha1.ImageRequest, *apisv1alpha1.ImageRequestList]
}

// newImageRequests returns a ImageRequests
func newImageRequests(c *ClabernetesV1alpha1Client, namespace string) *imageRequests {
	return &imageRequests{
		gentype.NewClientWithList[*apisv1alpha1.ImageRequest, *apisv1alpha1.ImageRequestList](
			"imagerequests",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *apisv1alpha1.ImageRequest { return &apisv1alpha1.ImageRequest{} },
			func() *apisv1alpha1.ImageRequestList { return &apisv1alpha1.ImageRequestList{} },
		),
	}
}
