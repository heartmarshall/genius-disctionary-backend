-- +goose Up
CREATE TABLE enrichment_queue (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ref_entry_id   UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    status         TEXT NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    priority       INT NOT NULL DEFAULT 0,
    error_message  TEXT,
    requested_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ux_enrichment_queue_ref_entry ON enrichment_queue(ref_entry_id);
CREATE INDEX ix_enrichment_queue_status ON enrichment_queue(status, priority DESC, requested_at);

-- +goose Down
DROP TABLE IF EXISTS enrichment_queue;
