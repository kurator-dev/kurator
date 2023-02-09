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

package infra

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
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/logger"
	"sigs.k8s.io/cluster-api-provider-aws/v2/util/system"
	capav1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"

	infrav1 "kurator.dev/kurator/pkg/apis/infra/v1alpha1"
	"kurator.dev/kurator/pkg/controllers/scope"
	"kurator.dev/kurator/pkg/controllers/template"
	"kurator.dev/kurator/pkg/typemeta"
)

func (r *ClusterController) reconcileAWS(ctx context.Context, infraCluster *infrav1.Cluster) (ctrl.Result, error) {
	ctxLogger := logger.FromContext(ctx)

	clusterCreds, err := r.reconcileAWSCreds(ctx, infraCluster)
	if err != nil {
		conditions.MarkFalse(infraCluster, infrav1.CredentialsReadyCondition, infrav1.ClusterCredentialsProvisioningFailedReason,
			capav1.ConditionSeverityError, err.Error())
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile cluster credentials")
	}
	conditions.MarkTrue(infraCluster, infrav1.CredentialsReadyCondition)

	if res, err := r.reconcileAWSIAMProfile(ctx, infraCluster, clusterCreds); err != nil {
		conditions.MarkFalse(infraCluster, infrav1.IAMProfileReadyCondition, infrav1.IAMProfileProvisioningFailedReason,
			capav1.ConditionSeverityError, err.Error())
		return res, errors.Wrapf(err, "failed to reconcile IAM profile")
	}
	conditions.MarkTrue(infraCluster, infrav1.IAMProfileReadyCondition)

	// TODO: ensure IRSA
	// TODO: support VPCConfig

	ctxLogger.Info("reconciling Cluster API resources")
	if _, err := r.reconcileAWSClusterAPIResources(ctx, infraCluster, clusterCreds); err != nil {
		conditions.MarkFalse(infraCluster, infrav1.ClusterAPIResourceReadyCondition, infrav1.ClusterAPIResourceProvisioningFailedReason,
			capav1.ConditionSeverityError, err.Error())
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile Cluster API resources")
	}

	ctxLogger.Info("reconciling CNI resources")
	if _, err := r.reconcileCNI(ctx, infraCluster); err != nil {
		conditions.MarkFalse(infraCluster, infrav1.ClusterAPIResourceReadyCondition, infrav1.ClusterAPIResourceProvisioningFailedReason,
			capav1.ConditionSeverityError, err.Error())
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile CNI resources")
	}
	conditions.MarkTrue(infraCluster, infrav1.ClusterAPIResourceReadyCondition)

	return ctrl.Result{}, nil
}

func (r *ClusterController) reconcileAWSClusterAPIResources(ctx context.Context, infraCluster *infrav1.Cluster, clusterCreds *corev1.Secret) (ctrl.Result, error) {
	c := scope.NewCluster(infraCluster, clusterCreds.Name)
	_, err := r.reconcileAWSCluster(ctx, c)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile AWSCluster %s/%s", infraCluster.Namespace, infraCluster.Name)
	}

	b, err := template.RenderClusterAPIForAWS(c)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to render Cluster API resources")
	}
	if _, err := patchResources(ctx, b); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to apply Cluster API resources")
	}

	return ctrl.Result{}, nil
}

func (r *ClusterController) reconcileAWSCluster(ctx context.Context, scopeCluster *scope.Cluster) (*awsinfrav1.AWSCluster, error) {
	awsCluster := &awsinfrav1.AWSCluster{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: scopeCluster.Namespace, Name: scopeCluster.Name}, awsCluster); err != nil {
		if apierrors.IsNotFound(err) {
			// create AWSCluster
			awsCluster.Name = scopeCluster.Name
			awsCluster.Namespace = scopeCluster.Namespace
			awsCluster.Spec.Region = scopeCluster.Region
			awsCluster.Spec.IdentityRef = &awsinfrav1.AWSIdentityReference{
				Kind: awsinfrav1.ClusterStaticIdentityKind,
				Name: scopeCluster.Credential,
			}

			if err := r.Create(ctx, awsCluster); err != nil {
				return nil, errors.Wrapf(err, "failed to create AWSCluster %s/%s", scopeCluster.Namespace, scopeCluster.Name)
			}

			return awsCluster, nil
		}

		return awsCluster, errors.Wrapf(err, "failed to get AWSCluster %s/%s", scopeCluster.Namespace, scopeCluster.Name)
	}

	awsCluster.Spec.IdentityRef = &awsinfrav1.AWSIdentityReference{
		Kind: awsinfrav1.ClusterStaticIdentityKind,
		Name: scopeCluster.Credential,
	}

	if err := r.Update(ctx, awsCluster); err != nil {
		return nil, errors.Wrapf(err, "failed to update AWSCluster %s/%s", scopeCluster.Namespace, scopeCluster.Name)
	}

	return awsCluster, nil
}

func (r *ClusterController) reconcileAWSCreds(ctx context.Context, infraCluster *infrav1.Cluster) (*corev1.Secret, error) {
	credSecret := &corev1.Secret{}
	nn := types.NamespacedName{
		Namespace: infraCluster.Namespace,
		Name:      infraCluster.Spec.Credential.SecretRef,
	}
	if err := r.Get(ctx, nn, credSecret); err != nil {
		return nil, errors.Wrapf(err, "failed to get secret %s", nn.String())
	}

	// TODO: verify secret is valid

	clusterSecret, err := r.reconcileAWSClusterSecret(ctx, infraCluster, credSecret)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile cluster secret")
	}

	if err := r.reconcileAWSIdentity(ctx, infraCluster, clusterSecret); err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile IAM profile")
	}

	return clusterSecret, nil
}

func (r *ClusterController) reconcileAWSClusterSecret(ctx context.Context, infraCluster *infrav1.Cluster, credSecret *corev1.Secret) (*corev1.Secret, error) {
	ctlNamespace := system.GetManagerNamespace()
	clusterCreds := &corev1.Secret{}
	secretName := r.NameGenerator.Generate(types.NamespacedName{
		Namespace: infraCluster.Namespace,
		Name:      infraCluster.Name,
	})
	nn := types.NamespacedName{Namespace: ctlNamespace, Name: secretName}
	if err := r.Get(ctx, nn, clusterCreds); err != nil {
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
				Labels:    clusterMatchingLabels(infraCluster),
			},
			StringData: credSecret.StringData,
			Data:       credSecret.Data,
		}
		if err := r.Create(ctx, clusterCreds); err != nil {
			return nil, errors.Wrapf(err, "failed to create secret %s", nn.String())
		}
	} else {
		clusterCreds.StringData = credSecret.StringData
		clusterCreds.Data = credSecret.Data
		if err := r.Update(ctx, clusterCreds); err != nil {
			return nil, errors.Wrapf(err, "failed to update secret %s", nn.String())
		}
	}

	return clusterCreds, nil
}

func (r *ClusterController) reconcileAWSIdentity(ctx context.Context, infraCluster *infrav1.Cluster, clusterSecret *corev1.Secret) error {
	awsIdentity := &awsinfrav1.AWSClusterStaticIdentity{}
	nn := types.NamespacedName{Name: clusterSecret.Name}
	if err := r.Get(ctx, nn, awsIdentity); err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to get AWSIdentity %s", nn.String())
		}

		awsIdentity = &awsinfrav1.AWSClusterStaticIdentity{
			TypeMeta: typemeta.AWSClusterStaticIdentity,
			ObjectMeta: metav1.ObjectMeta{
				Name:   clusterSecret.Name,
				Labels: clusterMatchingLabels(infraCluster),
			},
			Spec: awsinfrav1.AWSClusterStaticIdentitySpec{
				AWSClusterIdentitySpec: awsinfrav1.AWSClusterIdentitySpec{
					AllowedNamespaces: &awsinfrav1.AllowedNamespaces{
						NamespaceList: []string{infraCluster.Namespace},
					},
				},
				SecretRef: clusterSecret.Name,
			},
		}

		if err := r.Create(ctx, awsIdentity); err != nil {
			return errors.Wrapf(err, "failed to create AWSClusterStaticIdentity %s", nn.String())
		}
	} else {
		awsIdentity.Spec = awsinfrav1.AWSClusterStaticIdentitySpec{
			AWSClusterIdentitySpec: awsinfrav1.AWSClusterIdentitySpec{
				AllowedNamespaces: &awsinfrav1.AllowedNamespaces{
					NamespaceList: []string{infraCluster.Namespace},
				},
			},
			SecretRef: clusterSecret.Name,
		}

		if err := r.Update(ctx, awsIdentity); err != nil {
			return errors.Wrapf(err, "failed to update AWSClusterStaticIdentity %s", nn.String())
		}
	}

	return nil
}

func (r *ClusterController) reconcileAWSIAMProfile(ctx context.Context, infraCluster *infrav1.Cluster, credSecret *corev1.Secret) (ctrl.Result, error) {
	iamTpl := capabootstrap.NewTemplate()
	if infraCluster.Spec.PodIdentity.Enabled {
		iamTpl.Spec.S3Buckets.Enable = true
	}

	awsCfg := generateAWSConfig(ctx, infraCluster.Spec.Region, credSecret)
	s, err := session.NewSession(awsCfg)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to create AWS session")
	}

	cfnSvc := cloudformation.NewService(awssdkcfn.New(s))
	if err := cfnSvc.ReconcileBootstrapStack(awsbootstrapv1.DefaultStackName, *iamTpl.RenderCloudFormation(), nil); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile IAM bootstrap stack")
	}

	return ctrl.Result{}, nil
}

func (r *ClusterController) reconcileDeleteAWS(ctx context.Context, infraCluster *infrav1.Cluster) error {
	// AWSClusterStaticIdentitySpec is not namespaced, so we need to delete the identity by listing all of them matching the cluster labels
	awsIdentities := &awsinfrav1.AWSClusterStaticIdentityList{}
	if err := r.List(ctx, awsIdentities, clusterMatchingLabels(infraCluster)); err != nil {
		return errors.Wrapf(err, "failed to list AWSClusterStaticIdentity")
	}

	for _, identity := range awsIdentities.Items {
		if err := r.Delete(ctx, &identity); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}

			return errors.Wrapf(err, "failed to delete AWSClusterStaticIdentity %s", identity.Name)
		}
	}

	return nil
}
