-- +goose Up
CREATE TABLE ref_word_relations (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_entry_id  UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    target_entry_id  UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    relation_type    TEXT NOT NULL,
    source_slug      TEXT NOT NULL REFERENCES ref_data_sources(slug),
    created_at       TIMESTAMPTZ DEFAULT now(),

    CONSTRAINT chk_ref_word_relations_type
        CHECK (relation_type IN ('synonym', 'hypernym', 'antonym', 'derived')),

    CONSTRAINT uq_ref_word_relations
        UNIQUE (source_entry_id, target_entry_id, relation_type)
);

-- +goose Down
DROP TABLE IF EXISTS ref_word_relations;
