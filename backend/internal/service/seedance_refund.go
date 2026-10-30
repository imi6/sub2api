package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

// RefundVideoUsage reverses a prepaid Seedance video charge when the task reaches
// a terminal failure status. It:
//  1. Zeroes actual_cost/total_cost on the usage_log row and records refunded_amount.
//  2. Credits the user's balance by the refunded amount.
//  3. Decrements the Redis platform-quota counter by the refunded amount.
//  4. Invalidates the user's cached balance so the next eligibility check is fresh.
//
// No-op when the row was already refunded or never charged (amount == 0).
func (s *OpenAIGatewayService) RefundVideoUsage(ctx context.Context, stableRequestID string, userID, apiKeyID int64, platform string) {
	if s == nil || stableRequestID == "" || userID <= 0 || apiKeyID <= 0 {
		return
	}
	log := logger.L().With(
		zap.String("component", "service.seedance_refund"),
		zap.Int64("user_id", userID),
		zap.Int64("api_key_id", apiKeyID),
		zap.String("request_id", stableRequestID),
	)

	// 1. Mark usage row as refunded; get the refunded amount.
	amount, err := s.usageLogRepo.MarkVideoUsageRefunded(ctx, stableRequestID, apiKeyID)
	if err != nil {
		log.Error("seedance_refund.mark_refunded_failed", zap.Error(err))
		return
	}
	if amount <= 0 {
		// Row already refunded or never charged — nothing to reverse.
		return
	}

	log = log.With(zap.Float64("refunded_amount", amount))

	// 2. Credit the user's balance.
	if s.userRepo != nil {
		if _, err := s.userRepo.AdjustBalance(ctx, userID, amount); err != nil {
			log.Error("seedance_refund.adjust_balance_failed", zap.Error(err))
			// Continue: partial reversal beats full loss.
		}
	}

	// 3. Decrement platform-quota Redis counter.
	if s.billingCacheService != nil && platform != "" {
		s.billingCacheService.DecrementUserPlatformQuotaUsage(userID, platform, amount)
	}

	// 4. Invalidate cached balance so the next request sees the updated value.
	if s.billingCacheService != nil {
		if err := s.billingCacheService.InvalidateUserBalance(ctx, userID); err != nil {
			log.Warn("seedance_refund.invalidate_balance_failed", zap.Error(err))
		}
	}

	log.Info("seedance_refund.completed")
}
