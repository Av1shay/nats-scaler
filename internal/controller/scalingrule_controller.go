/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"
	scalingv1 "github.com/Av1shay/nats-scaler/api/v1"
	"github.com/Av1shay/nats-scaler/internal/nats"
	internalTypes "github.com/Av1shay/nats-scaler/internal/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

const (
	// TODO make configurable
	errRequeueIntervalLong  = time.Minute
	errRequeueIntervalShort = 10 * time.Second
)

type Scaler interface {
	ReconcileScale(ctx context.Context, client client.Client, nn types.NamespacedName, spec internalTypes.ScalerParams, pendings int) error
}

// ScalingRuleReconciler reconciles a ScalingRule object
type ScalingRuleReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NatsService *nats.Service
	Scaler      Scaler
}

// +kubebuilder:rbac:groups=scaling.my.domain,resources=scalingrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scaling.my.domain,resources=scalingrules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scaling.my.domain,resources=scalingrules/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *ScalingRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	var rule scalingv1.ScalingRule
	if err := r.Get(ctx, req.NamespacedName, &rule); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// if the resource deleted, we don't need to reconcile it again
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to get ScalingRule", "retryIn", errRequeueIntervalLong)
		return ctrl.Result{RequeueAfter: errRequeueIntervalLong}, nil
	}
	if err := validateScalingRuleSpec(rule.Spec); err != nil {
		logger.Error(err, "invalid ScalingRule spec", "retryIn", errRequeueIntervalLong)
		return ctrl.Result{RequeueAfter: errRequeueIntervalLong}, nil
	}

	pendings, err := r.NatsService.GetPendingMessages(ctx, rule.Spec.NatsMonitoringURL, rule.Spec.StreamName, rule.Spec.ConsumerName)
	if err != nil {
		logger.Error(err, "failed to get pending messages from NATS", "retryIn", errRequeueIntervalShort)
		return ctrl.Result{RequeueAfter: errRequeueIntervalShort}, nil
	}

	if err := r.Scaler.ReconcileScale(ctx, r.Client, types.NamespacedName{
		Name:      rule.Spec.DeploymentName,
		Namespace: rule.Spec.Namespace,
	}, internalTypes.ScalerParams{
		MinReplicas:        rule.Spec.MinReplicas,
		MaxReplicas:        rule.Spec.MaxReplicas,
		ScaleUpThreshold:   rule.Spec.ScaleUpThreshold,
		ScaleDownThreshold: rule.Spec.ScaleDownThreshold,
	}, pendings); err != nil {
		logger.Error(err, "failed to get pending messages from NATS", "retryIn", errRequeueIntervalShort)
		return ctrl.Result{RequeueAfter: errRequeueIntervalShort}, nil
	}

	return ctrl.Result{RequeueAfter: time.Duration(rule.Spec.PollIntervalSeconds) * time.Second}, nil
}

// ScalingRule runtime validation for spec, in real-world we will do this in a webhook
func validateScalingRuleSpec(spec scalingv1.ScalingRuleSpec) error {
	if spec.MinReplicas > spec.MaxReplicas {
		return fmt.Errorf("minReplicas (%d) must be less than or equal to maxReplicas (%d)", spec.MinReplicas, spec.MaxReplicas)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScalingRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1.ScalingRule{}).
		Named("scalingrule").
		Complete(r)
}
