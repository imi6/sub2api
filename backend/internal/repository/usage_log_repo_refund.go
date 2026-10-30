package repository

import (
	"context"
)

// MarkVideoUsageRefunded zeroes actual_cost/total_cost on the prepaid video row
// identified by (request_id, api_key_id) and saves the original amount in
// refunded_amount. Returns the refunded amount, or 0 if the row was already
// refunded or not found. Safe to call more than once (idempotent after first call).
func (r *usageLogRepository) MarkVideoUsageRefunded(ctx context.Context, requestID string, apiKeyID int64) (float64, error) {
	const q = `
		UPDATE usage_logs
		SET    refunded_amount = actual_cost,
		       actual_cost     = 0,
		       total_cost      = 0,
		       refunded_at     = NOW()
		WHERE  request_id  = $1
		  AND  api_key_id  = $2
		  AND  refunded_at IS NULL
		  AND  actual_cost  > 0
		RETURNING refunded_amount
	`
	rows, err := r.sql.QueryContext(ctx, q, requestID, apiKeyID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, nil // already refunded or not found
	}
	var amount float64
	if err := rows.Scan(&amount); err != nil {
		return 0, err
	}
	return amount, rows.Err()
}
