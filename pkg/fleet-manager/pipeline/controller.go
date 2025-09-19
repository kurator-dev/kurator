/*
Copyright 2022-2025 Kurator Authors.
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

package pipeline

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	pipelineapi "kurator.dev/kurator/pkg/apis/pipeline/v1alpha1"
	"kurator.dev/kurator/pkg/fleet-manager/pipeline/render"
	"kurator.dev/kurator/pkg/infra/util"
)

const (
	PipelineFinalizer   = "pipeline.kurator.dev"
	TektonPipelineLabel = "tekton.dev/pipeline"
	ChainCredentials    = "chain-credentials"
)

// PipelineManager reconciles a Pipeline object.
type PipelineManager struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (p *PipelineManager) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelineapi.Pipeline{}).
		WithOptions(options).
		Complete(p)
}

// Reconcile performs the reconciliation process for the Pipeline object.
func (p *PipelineManager) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)
	pipeline := &pipelineapi.Pipeline{}

	// Retrieve the pipeline object based on the request.
	if err := p.Client.Get(ctx, req.NamespacedName, pipeline); err != nil {
		log.Error(err, "failed to fetching pipeline")

		// Handle not found errors and requeue others.
		if apierrors.IsNotFound(err) {
			log.Info("Pipeline object not found", "pipeline", req)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Initialize a helper for patching the pipeline object at the end.
	patchHelper, err := patch.NewHelper(pipeline, p.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to init patch helper for pipeline %s", req.NamespacedName)
	}
	defer func() {
		patchOpts := []patch.Option{}
		if err := patchHelper.Patch(ctx, pipeline, patchOpts...); err != nil {
			log.Error(err, "error patching pipeline")
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch pipeline %s", req.NamespacedName)})
		}
	}()

	// Add finalizer if it's not present.
	if !controllerutil.ContainsFinalizer(pipeline, PipelineFinalizer) {
		controllerutil.AddFinalizer(pipeline, PipelineFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle pipeline deletion.
	if pipeline.GetDeletionTimestamp() != nil {
		return p.reconcileDeletePipeline(ctx, pipeline)
	}

	// Proceed with the main reconciliation logic.
	return p.reconcile(ctx, pipeline)
}

// reconcile contains the core logic for reconciling the Pipeline object.
func (p *PipelineManager) reconcile(ctx context.Context, pipeline *pipelineapi.Pipeline) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Reconcile Tekton RBAC resource.
	res, err := p.reconcileRBAC(ctx, pipeline)
	if err != nil || res.Requeue || res.RequeueAfter > 0 {
		log.Error(err, "Error creating Tekton tasks")
		return res, err
	}

	// Reconcile Tekton tasks.
	res, err = p.reconcileTasks(ctx, pipeline)
	if err != nil || res.Requeue || res.RequeueAfter > 0 {
		log.Error(err, "Error creating Tekton tasks")
		return res, err
	}

	// Reconcile Tekton pipeline.
	res, err = p.reconcilePipeline(ctx, pipeline)
	if err != nil || res.Requeue || res.RequeueAfter > 0 {
		return res, err
	}

	// Reconcile Tekton trigger.
	res, err = p.reconcileTrigger(ctx, pipeline)
	if err != nil || res.Requeue || res.RequeueAfter > 0 {
		return res, err
	}

	// Reconcile pipeline status.
	return p.reconcilePipelineStatus(ctx, pipeline)
}

// reconcileRBAC renders and syncs RBAC resources for the pipeline.
func (p *PipelineManager) reconcileRBAC(ctx context.Context, pipeline *pipelineapi.Pipeline) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Prepare RBAC configuration for the pipeline.
	rbacConfig := render.RBACConfig{
		PipelineName:      pipeline.Name,
		PipelineNamespace: pipeline.Namespace,
		OwnerReference:    render.GeneratePipelineOwnerRef(pipeline),
	}

	// Check if we need ChainCredentials
	if needChainCredentials(pipeline) {
		rbacConfig.ChainCredentialsName = ChainCredentials
	}

	// Render RBAC configuration.
	rbac, err := render.RenderRBAC(rbacConfig)
	if err != nil {
		log.Error(err, "Error rendering RBAC resources")
		return ctrl.Result{}, err
	}

	// Apply RBAC resources.
	if _, err := util.PatchResources(rbac); err != nil {
		log.Error(err, "Error applying RBAC resources")
		return ctrl.Result{}, errors.Wrapf(err, "failed to apply RBAC resources")
	}

	return ctrl.Result{}, nil
}

// reconcileTasks renders and syncs Tekton tasks for the pipeline.
func (p *PipelineManager) reconcileTasks(ctx context.Context, pipeline *pipelineapi.Pipeline) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Process each task in the pipeline.
	for _, task := range pipeline.Spec.Tasks {
		var err error
		if task.PredefinedTask != nil {
			err = p.createPredefinedTask(ctx, &task, pipeline)
		} else {
			err = p.createCustomTask(ctx, &task, pipeline)
		}
		if err != nil {
			log.Error(err, "Error creating task", "taskName", task.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// createPredefinedTask renders and syncs a predefined Tekton task.
func (p *PipelineManager) createPredefinedTask(ctx context.Context, task *pipelineapi.PipelineTask, pipeline *pipelineapi.Pipeline) error {
	log := ctrl.LoggerFrom(ctx)

	// Render the predefined task.
	taskResource, err := render.RenderPredefinedTaskWithPipeline(pipeline, task.PredefinedTask)
	if err != nil {
		log.Error(err, "Error rendering predefined task")
		return err
	}

	// Apply the task resources.
	if _, err := util.PatchResources(taskResource); err != nil {
		log.Error(err, "Error applying task resources")
		return errors.Wrapf(err, "failed to apply task resources")
	}

	return nil
}

// createCustomTask renders and syncs a custom Tekton task.
func (p *PipelineManager) createCustomTask(ctx context.Context, task *pipelineapi.PipelineTask, pipeline *pipelineapi.Pipeline) error {
	log := ctrl.LoggerFrom(ctx)

	// Render the custom task.
	taskResource, err := render.RenderCustomTaskWithPipeline(pipeline, task.Name, task.CustomTask)
	if err != nil {
		log.Error(err, "Error rendering custom task")
		return err
	}

	// Apply the task resources.
	if _, err := util.PatchResources(taskResource); err != nil {
		log.Error(err, "Error applying custom task resources")
		return errors.Wrapf(err, "failed to apply custom task resources")
	}

	return nil
}

// reconcilePipeline renders and syncs the Tekton pipeline.
func (p *PipelineManager) reconcilePipeline(ctx context.Context, pipeline *pipelineapi.Pipeline) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Render the pipeline.
	pipelineResource, err := render.RenderPipelineWithPipeline(pipeline)
	if err != nil {
		log.Error(err, "Error rendering the pipeline")
		return ctrl.Result{}, err
	}

	// Apply the pipeline resources.
	if _, err := util.PatchResources(pipelineResource); err != nil {
		log.Error(err, "Error applying pipeline resources")
		return ctrl.Result{}, errors.Wrapf(err, "failed to apply pipeline resources")
	}

	return ctrl.Result{}, nil
}

// reconcileTrigger renders and syncs the Tekton trigger.
func (p *PipelineManager) reconcileTrigger(ctx context.Context, pipeline *pipelineapi.Pipeline) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Render the trigger.
	triggerResource, err := render.RenderTriggerWithPipeline(pipeline)
	if err != nil {
		log.Error(err, "Error rendering the trigger")
		return ctrl.Result{}, err
	}

	// Apply the trigger resources.
	if _, err := util.PatchResources(triggerResource); err != nil {
		log.Error(err, "Error applying trigger resources")
		return ctrl.Result{}, errors.Wrapf(err, "failed to apply trigger resources")
	}

	return ctrl.Result{}, nil
}

// reconcilePipelineStatus updates the status of the pipeline.
func (p *PipelineManager) reconcilePipelineStatus(ctx context.Context, pipeline *pipelineapi.Pipeline) (ctrl.Result, error) {
	// Update event listener service name in the status.
	pipeline.Status.EventListenerServiceName = getListenerServiceName(pipeline)
	return ctrl.Result{}, nil
}

// reconcileDeletePipeline handles the deletion of a Pipeline object.
func (p *PipelineManager) reconcileDeletePipeline(ctx context.Context, pipeline *pipelineapi.Pipeline) (ctrl.Result, error) {
	// First, delete all Pods with the specific label. The other resource created by pipeline will be deleted with ownerRef
	if err := p.deleteAssociatedPods(ctx, pipeline.Namespace, pipeline.Name); err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting associated pods: %v", err)
	}

	// After successful deletion of Pods, remove the finalizer from the Pipeline.
	controllerutil.RemoveFinalizer(pipeline, PipelineFinalizer)

	return ctrl.Result{}, nil
}

// deleteAssociatedPods deletes all Pods in the same namespace as the Pipeline
// that have a label with key TektonPipelineLabel and value equal to pipelineName.
func (p *PipelineManager) deleteAssociatedPods(ctx context.Context, namespace, pipelineName string) error {
	// The Kurator pipeline name is equal to the Tekton pipeline name, so we can use MatchingLabels{TektonPipelineLabel: pipelineName}
	labelSelector := client.MatchingLabels{TektonPipelineLabel: pipelineName}
	var pods corev1.PodList

	// List all Pods in the same namespace with the specified label selector.
	if err := p.Client.List(ctx, &pods, client.InNamespace(namespace), labelSelector); err != nil {
		return fmt.Errorf("error listing pods: %v", err)
	}

	// Delete each found Pod.
	for _, pod := range pods.Items {
		// Delete the Pod using Foreground deletion policy.
		// This ensures that all dependent resources like PVCs are also deleted before the Pod itself is deleted.
		err := p.Client.Delete(ctx, &pod, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if err != nil && !apierrors.IsNotFound(err) {
			// If the error is not a NotFound error, return the error.
			return fmt.Errorf("error deleting pod %s: %v", pod.Name, err)
		}
	}

	return nil
}

// getListenerServiceName get the name of event listener service name. This naming way is origin from tekton controller.
// TODO: add associated link about tekton code for this naming
func getListenerServiceName(pipeline *pipelineapi.Pipeline) *string {
	serviceName := "el-" + pipeline.Name + "-listener"
	return &serviceName
}

// needChainCredentials show if this pipeline need user create Credentials. Currently, it will return true when we have predefined task named BuildPushImageContent
func needChainCredentials(pipeline *pipelineapi.Pipeline) bool {
	for _, task := range pipeline.Spec.Tasks {
		if task.CustomTask != nil {
			continue
		}
		if task.PredefinedTask.Name == render.BuildPushImageContent {
			return true
		}
	}
	return false
}
