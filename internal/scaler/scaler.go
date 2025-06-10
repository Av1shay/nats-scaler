package scaler

import (
	"context"
	"fmt"
	"sync"
	"time"

	internalTypes "github.com/Av1shay/nats-scaler/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultCooldown = 15 * time.Second
)

var (
	lastScaleMap = sync.Map{} // map[types.NamespacedName]time.Time
)

type RealScaler struct {
	cooldown time.Duration
}

type Option func(*RealScaler)

func WithCooldown(cooldown time.Duration) Option {
	return func(s *RealScaler) {
		s.cooldown = cooldown
	}
}

func NewScaler(options ...Option) *RealScaler {
	s := &RealScaler{cooldown: defaultCooldown}
	for _, opt := range options {
		opt(s)
	}
	return s
}

func (s *RealScaler) ReconcileScale(
	ctx context.Context,
	k8s client.Client,
	deployment types.NamespacedName,
	rule internalTypes.ScalerParams,
	pendingMsgs int,
) error {
	logger := logf.FromContext(ctx)
	now := time.Now()

	// make sure we are not scaling too aggressively
	if val, ok := lastScaleMap.Load(deployment); ok {
		if last, ok := val.(time.Time); ok && now.Sub(last) < s.cooldown {
			logger.Info(fmt.Sprintf("Cooldown in effect — skipping scaling. (pending: %d)", pendingMsgs))
			return nil
		}
	}

	var deploy appsv1.Deployment
	if err := k8s.Get(ctx, deployment, &deploy); err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	current := *deploy.Spec.Replicas
	desired := current

	if pendingMsgs > rule.ScaleUpThreshold && current < rule.MaxReplicas {
		desired = current + 1
		if desired > rule.MaxReplicas {
			desired = rule.MaxReplicas
		}
		logger.Info(fmt.Sprintf("Scaling up: %d → %d (pending: %d > %d)", current, desired, pendingMsgs, rule.ScaleUpThreshold))
	} else if pendingMsgs < rule.ScaleDownThreshold && current > rule.MinReplicas {
		desired = current - 1
		if desired < rule.MinReplicas {
			desired = rule.MinReplicas
		}
		logger.Info(fmt.Sprintf("Scaling down: %d → %d (pending: %d < %d)", current, desired, pendingMsgs, rule.ScaleDownThreshold))
	}

	if desired != current {
		deploy.Spec.Replicas = &desired
		if err := k8s.Update(ctx, &deploy); err != nil {
			return fmt.Errorf("failed to update deployment: %w", err)
		}
		lastScaleMap.Store(deployment, now)
	}

	return nil
}
