-- 异步视频任务失败退款：原用量行的 actual_cost/total_cost 清零并记录退款时间与原扣费金额，
-- 统计查询无需改动即可排除已退款金额。
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS refunded_at TIMESTAMPTZ;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS refunded_amount NUMERIC(20,10);
