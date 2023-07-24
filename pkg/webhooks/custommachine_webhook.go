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

package webhooks

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"kurator.dev/kurator/pkg/apis/infra/v1alpha1"
)

var _ webhook.CustomValidator = &CustomMachineWebhook{}

type CustomMachineWebhook struct {
	Client client.Reader
}

func (wh *CustomMachineWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.CustomMachine{}).
		WithValidator(wh).
		Complete()
}

func (wh *CustomMachineWebhook) ValidateCreate(_ context.Context, obj runtime.Object) error {
	in, ok := obj.(*v1alpha1.CustomMachine)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a CustomMachine but got a %T", obj))
	}

	return wh.validate(in)
}

func (wh *CustomMachineWebhook) validate(in *v1alpha1.CustomMachine) error {
	var allErrs field.ErrorList

	if len(in.Spec.Master) == 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "master"), in.Spec.Master,
			"at least one master must be configured"))
	} else if len(in.Spec.Master)%2 == 0 {
		// etcd nodes must be set to an odd number, see https://github.com/kubernetes-sigs/kubespray/blob/0405af11077bc271529f9eca790a7dac4edf3891/docs/nodes.md
		// we do not have a dedicated etcd configuration, so we are using the master node as the etcd node
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "master"), len(in.Spec.Master),
			"the number of master nodes need to be set to an odd number due to the restrictions of etcd cluster."))
	}

	if len(in.Spec.Nodes) == 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "nodes"), in.Spec.Nodes,
			"at least one node must be configured"))
	}

	allErrs = append(allErrs, validateMachine(in.Spec.Master, field.NewPath("spec", "master"))...)
	allErrs = append(allErrs, validateMachine(in.Spec.Nodes, field.NewPath("spec", "node"))...)

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(v1alpha1.SchemeGroupVersion.WithKind("CustomMachine").GroupKind(), in.Name, allErrs)
	}

	return nil
}

func validateMachine(machineArr []v1alpha1.Machine, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	for i, machine := range machineArr {
		if machine.PrivateIP == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("privateIP"), "must be set"))
		} else {
			allErrs = append(allErrs, validateIP(machine.PrivateIP, fldPath.Index(i).Child("privateIP"))...)
		}
		if machine.PublicIP == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("publicIP"), "must be set"))
		} else {
			allErrs = append(allErrs, validateIP(machine.PublicIP, fldPath.Index(i).Child("publicIP"))...)
		}
	}

	return allErrs
}

func (wh *CustomMachineWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	_, ok := oldObj.(*v1alpha1.CustomMachine)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a CustomMachine but got a %T", oldObj))
	}

	newCustomMachine, ok := newObj.(*v1alpha1.CustomMachine)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a CustomMachine but got a %T", newObj))
	}

	return wh.validate(newCustomMachine)
}

func (wh *CustomMachineWebhook) ValidateDelete(_ context.Context, obj runtime.Object) error {
	return nil
}
