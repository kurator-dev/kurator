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

package infraprovider

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	awssdkcfn "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	awsinfrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	awsbootstrapv1 "sigs.k8s.io/cluster-api-provider-aws/v2/cmd/clusterawsadm/api/bootstrap/v1beta1"
	capabootstrap "sigs.k8s.io/cluster-api-provider-aws/v2/cmd/clusterawsadm/cloudformation/bootstrap"
	cloudformation "sigs.k8s.io/cluster-api-provider-aws/v2/cmd/clusterawsadm/cloudformation/service"
	"sigs.k8s.io/cluster-api-provider-aws/v2/util/system"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	"kurator.dev/kurator/pkg/infra/scope"
	"kurator.dev/kurator/pkg/infra/template"
	"kurator.dev/kurator/pkg/infra/util"
	"kurator.dev/kurator/pkg/typemeta"
)

type Provider interface {
	Reconcile(ctx context.Context, cluster *infrav1.Cluster) error
	Clean(ctx context.Context, cluster *infrav1.Cluster) error
}

func NewProvider(kube client.Client, scope *scope.Cluster) Provider {
	return &AWSProvider{
		Kube:  kube,
		scope: scope,
	}
}

var _ Provider = &AWSProvider{}

type AWSProvider struct {
	Kube  client.Client
	scope *scope.Cluster
}

func (p *AWSProvider) Reconcile(ctx context.Context, infraCluster *infrav1.Cluster) error {
	clusterCreds, err := p.reconcileAWSCreds(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to reconcile cluster credentials")
	}

	if err := p.reconcileAWSIAMProfile(ctx, clusterCreds); err != nil {
		return errors.Wrapf(err, "failed to reconcile IAM profile")
	}

	// TODO: ensure IRSA

	if _, err := p.reconcileAWSClusterAPIResources(ctx, clusterCreds); err != nil {
		return errors.Wrapf(err, "failed to reconcile Cluster API resources")
	}

	// TODO: update VPC.Name if needed

	return nil
}

func (p *AWSProvider) Clean(ctx context.Context, cluster *infrav1.Cluster) error {
	// AWSClusterStaticIdentitySpec is not namespaced, so we need to delete the identity by listing all of them matching the cluster labels
	awsIdentities := &awsinfrav1.AWSClusterStaticIdentityList{}
	if err := p.Kube.List(ctx, awsIdentities, util.ClusterMatchingLabels(cluster)); err != nil {
		return errors.Wrapf(err, "failed to list AWSClusterStaticIdentity")
	}

	for _, identity := range awsIdentities.Items {
		if err := p.Kube.Delete(ctx, &identity); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}

			return errors.Wrapf(err, "failed to delete AWSClusterStaticIdentity %s", identity.Name)
		}
	}

	return nil
}

func (p *AWSProvider) reconcileAWSClusterAPIResources(ctx context.Context, clusterCreds *corev1.Secret) (ctrl.Result, error) {
	_, err := p.reconcileAWSCluster(ctx)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile AWSCluster %s/%s", p.scope.Namespace, p.scope.Name)
	}

	b, err := template.RenderClusterAPIForAWS(p.scope)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to render Cluster API resources")
	}
	if _, err := util.PatchResources(b); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to apply Cluster API resources")
	}

	return ctrl.Result{}, nil
}

func (p *AWSProvider) reconcileAWSCluster(ctx context.Context) (*awsinfrav1.AWSCluster, error) {
	scopeCluster := p.scope

	awsCluster := &awsinfrav1.AWSCluster{}
	if err := p.Kube.Get(ctx, types.NamespacedName{Namespace: scopeCluster.Namespace, Name: scopeCluster.Name}, awsCluster); err != nil {
		if apierrors.IsNotFound(err) {
			// create AWSCluster
			awsCluster.Name = scopeCluster.Name
			awsCluster.Namespace = scopeCluster.Namespace
			awsCluster.Spec = awsinfrav1.AWSClusterSpec{
				Region: scopeCluster.Region,
				IdentityRef: &awsinfrav1.AWSIdentityReference{
					Kind: awsinfrav1.ClusterStaticIdentityKind,
					Name: scopeCluster.SecretName(),
				},
				NetworkSpec: awsinfrav1.NetworkSpec{
					VPC: awsinfrav1.VPCSpec{
						CidrBlock: scopeCluster.VpcCIDR,
					},
				},
			}
			if err := p.Kube.Create(ctx, awsCluster); err != nil {
				return nil, errors.Wrapf(err, "failed to create AWSCluster %s/%s", scopeCluster.Namespace, scopeCluster.Name)
			}

			return awsCluster, nil
		}

		return awsCluster, errors.Wrapf(err, "failed to get AWSCluster %s/%s", scopeCluster.Namespace, scopeCluster.Name)
	}

	awsCluster.Spec.NetworkSpec.VPC.CidrBlock = scopeCluster.VpcCIDR
	awsCluster.Spec.IdentityRef = &awsinfrav1.AWSIdentityReference{
		Kind: awsinfrav1.ClusterStaticIdentityKind,
		Name: scopeCluster.SecretName(),
	}

	if err := p.Kube.Update(ctx, awsCluster); err != nil {
		return nil, errors.Wrapf(err, "failed to update AWSCluster %s/%s", scopeCluster.Namespace, scopeCluster.Name)
	}

	return awsCluster, nil
}

func (p *AWSProvider) reconcileAWSCreds(ctx context.Context) (*corev1.Secret, error) {
	credSecret := &corev1.Secret{}
	nn := types.NamespacedName{
		Namespace: p.scope.Namespace,
		Name:      p.scope.CredentialSecretRef,
	}
	if err := p.Kube.Get(ctx, nn, credSecret); err != nil {
		return nil, errors.Wrapf(err, "failed to get secret %s", nn.String())
	}

	// TODO: verify secret is valid

	clusterSecret, err := p.reconcileAWSClusterSecret(ctx, credSecret)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile cluster secret")
	}

	if err := p.reconcileAWSIdentity(ctx, clusterSecret); err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile IAM profile")
	}

	return clusterSecret, nil
}

func (p *AWSProvider) reconcileAWSClusterSecret(ctx context.Context, credSecret *corev1.Secret) (*corev1.Secret, error) {
	ctlNamespace := system.GetManagerNamespace()
	clusterCreds := &corev1.Secret{}
	secretName := p.scope.SecretName()

	nn := types.NamespacedName{Namespace: ctlNamespace, Name: secretName}
	if err := p.Kube.Get(ctx, nn, clusterCreds); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "failed to get secret %s", nn.String())
		}

		// AWS provider will set OwnerReference to secret referenced by AWSClusterStaticIdentity, so we don't need to set it here.
		// see more details here: https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/main/pkg/cloud/scope/session.go#L339
		clusterCreds = &corev1.Secret{
			TypeMeta: typemeta.Secret,
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ctlNamespace, // provider always use system namespace
				Name:      secretName,
			},
			StringData: credSecret.StringData,
			Data:       credSecret.Data,
		}
		if err := p.Kube.Create(ctx, clusterCreds); err != nil {
			return nil, errors.Wrapf(err, "failed to create secret %s", nn.String())
		}
	} else {
		clusterCreds.StringData = credSecret.StringData
		clusterCreds.Data = credSecret.Data
		if err := p.Kube.Update(ctx, clusterCreds); err != nil {
			return nil, errors.Wrapf(err, "failed to update secret %s", nn.String())
		}
	}

	return clusterCreds, nil
}

func (p *AWSProvider) reconcileAWSIdentity(ctx context.Context, clusterSecret *corev1.Secret) error {
	awsIdentity := &awsinfrav1.AWSClusterStaticIdentity{}
	nn := types.NamespacedName{Name: clusterSecret.Name}
	if err := p.Kube.Get(ctx, nn, awsIdentity); err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to get AWSIdentity %s", nn.String())
		}

		awsIdentity = &awsinfrav1.AWSClusterStaticIdentity{
			TypeMeta: typemeta.AWSClusterStaticIdentity,
			ObjectMeta: metav1.ObjectMeta{
				Name:   clusterSecret.Name,
				Labels: p.scope.MatchingLabels(),
			},
			Spec: awsinfrav1.AWSClusterStaticIdentitySpec{
				AWSClusterIdentitySpec: awsinfrav1.AWSClusterIdentitySpec{
					AllowedNamespaces: &awsinfrav1.AllowedNamespaces{
						NamespaceList: []string{p.scope.Namespace},
					},
				},
				SecretRef: clusterSecret.Name,
			},
		}

		if err := p.Kube.Create(ctx, awsIdentity); err != nil {
			return errors.Wrapf(err, "failed to create AWSClusterStaticIdentity %s", nn.String())
		}
	} else {
		awsIdentity.Spec = awsinfrav1.AWSClusterStaticIdentitySpec{
			AWSClusterIdentitySpec: awsinfrav1.AWSClusterIdentitySpec{
				AllowedNamespaces: &awsinfrav1.AllowedNamespaces{
					NamespaceList: []string{p.scope.Namespace},
				},
			},
			SecretRef: clusterSecret.Name,
		}

		if err := p.Kube.Update(ctx, awsIdentity); err != nil {
			return errors.Wrapf(err, "failed to update AWSClusterStaticIdentity %s", nn.String())
		}
	}

	return nil
}

func (p *AWSProvider) reconcileAWSIAMProfile(ctx context.Context, credSecret *corev1.Secret) error {
	iamTpl := capabootstrap.NewTemplate()
	if p.scope.EnablePodIdentity {
		iamTpl.Spec.S3Buckets.Enable = true
	}

	awsCfg := util.AWSConfig(p.scope.Region, credSecret)
	s, err := session.NewSession(awsCfg)
	if err != nil {
		return errors.Wrapf(err, "failed to create AWS session")
	}

	cfnSvc := cloudformation.NewService(awssdkcfn.New(s))
	if err := cfnSvc.ReconcileBootstrapStack(awsbootstrapv1.DefaultStackName, *iamTpl.RenderCloudFormation(), nil); err != nil {
		return errors.Wrapf(err, "failed to reconcile IAM bootstrap stack")
	}

	return nil
}
