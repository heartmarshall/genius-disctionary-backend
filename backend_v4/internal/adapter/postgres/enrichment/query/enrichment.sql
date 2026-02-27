-- name: Enqueue :exec
INSERT INTO enrichment_queue (ref_entry_id, priority)
VALUES ($1, $2)
ON CONFLICT (ref_entry_id)
DO UPDATE SET priority = enrichment_queue.priority + 1,
              requested_at = now()
WHERE enrichment_queue.status IN ('pending', 'failed');

-- name: ClaimBatch :many
UPDATE enrichment_queue
SET status = 'processing'
WHERE id IN (
    SELECT id FROM enrichment_queue
    WHERE status = 'pending'
    ORDER BY priority DESC, requested_at
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
RETURNING id, ref_entry_id, status, priority, error_message, requested_at, processed_at, created_at;

-- name: MarkDone :exec
UPDATE enrichment_queue SET status = 'done', processed_at = now(), error_message = NULL
WHERE ref_entry_id = $1;

-- name: MarkFailed :exec
UPDATE enrichment_queue SET status = 'failed', processed_at = now(), error_message = $2
WHERE ref_entry_id = $1;

-- name: GetStats :one
SELECT
    count(*) FILTER (WHERE status = 'pending')    AS pending,
    count(*) FILTER (WHERE status = 'processing') AS processing,
    count(*) FILTER (WHERE status = 'done')        AS done,
    count(*) FILTER (WHERE status = 'failed')      AS failed,
    count(*)                                        AS total
FROM enrichment_queue;

-- name: List :many
SELECT id, ref_entry_id, status, priority, error_message, requested_at, processed_at, created_at
FROM enrichment_queue
WHERE ($1::text = '' OR status = $1)
ORDER BY priority DESC, requested_at
LIMIT $2 OFFSET $3;

-- name: RetryAllFailed :execrows
UPDATE enrichment_queue
SET status = 'pending', error_message = NULL, processed_at = NULL
WHERE status = 'failed';

-- name: ResetProcessing :execrows
UPDATE enrichment_queue
SET status = 'pending'
WHERE status = 'processing';
