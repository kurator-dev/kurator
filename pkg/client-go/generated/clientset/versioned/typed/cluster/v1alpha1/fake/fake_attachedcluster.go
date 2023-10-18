/*
Copyright Kurator Authors.

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

package fake

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1alpha1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
)

// FakeAttachedClusters implements AttachedClusterInterface
type FakeAttachedClusters struct {
	Fake *FakeClusterV1alpha1
	ns   string
}

var attachedclustersResource = v1alpha1.SchemeGroupVersion.WithResource("attachedclusters")

var attachedclustersKind = v1alpha1.SchemeGroupVersion.WithKind("AttachedCluster")

// Get takes name of the attachedCluster, and returns the corresponding attachedCluster object, and an error if there is any.
func (c *FakeAttachedClusters) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.AttachedCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(attachedclustersResource, c.ns, name), &v1alpha1.AttachedCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AttachedCluster), err
}

// List takes label and field selectors, and returns the list of AttachedClusters that match those selectors.
func (c *FakeAttachedClusters) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.AttachedClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(attachedclustersResource, attachedclustersKind, c.ns, opts), &v1alpha1.AttachedClusterList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.AttachedClusterList{ListMeta: obj.(*v1alpha1.AttachedClusterList).ListMeta}
	for _, item := range obj.(*v1alpha1.AttachedClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested attachedClusters.
func (c *FakeAttachedClusters) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(attachedclustersResource, c.ns, opts))

}

// Create takes the representation of a attachedCluster and creates it.  Returns the server's representation of the attachedCluster, and an error, if there is any.
func (c *FakeAttachedClusters) Create(ctx context.Context, attachedCluster *v1alpha1.AttachedCluster, opts v1.CreateOptions) (result *v1alpha1.AttachedCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(attachedclustersResource, c.ns, attachedCluster), &v1alpha1.AttachedCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AttachedCluster), err
}

// Update takes the representation of a attachedCluster and updates it. Returns the server's representation of the attachedCluster, and an error, if there is any.
func (c *FakeAttachedClusters) Update(ctx context.Context, attachedCluster *v1alpha1.AttachedCluster, opts v1.UpdateOptions) (result *v1alpha1.AttachedCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(attachedclustersResource, c.ns, attachedCluster), &v1alpha1.AttachedCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AttachedCluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAttachedClusters) UpdateStatus(ctx context.Context, attachedCluster *v1alpha1.AttachedCluster, opts v1.UpdateOptions) (*v1alpha1.AttachedCluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(attachedclustersResource, "status", c.ns, attachedCluster), &v1alpha1.AttachedCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AttachedCluster), err
}

// Delete takes name of the attachedCluster and deletes it. Returns an error if one occurs.
func (c *FakeAttachedClusters) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(attachedclustersResource, c.ns, name, opts), &v1alpha1.AttachedCluster{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAttachedClusters) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(attachedclustersResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.AttachedClusterList{})
	return err
}

// Patch applies the patch and returns the patched attachedCluster.
func (c *FakeAttachedClusters) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.AttachedCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(attachedclustersResource, c.ns, name, pt, data, subresources...), &v1alpha1.AttachedCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AttachedCluster), err
}
