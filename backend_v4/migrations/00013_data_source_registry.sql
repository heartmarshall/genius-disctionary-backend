-- +goose Up
CREATE TABLE ref_data_sources (
    slug           TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    description    TEXT,
    source_type    TEXT NOT NULL,
    is_active      BOOLEAN DEFAULT true,
    dataset_version TEXT,
    created_at     TIMESTAMPTZ DEFAULT now(),
    updated_at     TIMESTAMPTZ DEFAULT now(),

    CONSTRAINT chk_ref_data_sources_type
        CHECK (source_type IN ('definitions', 'translations', 'pronunciations', 'examples', 'relations', 'metadata'))
);

INSERT INTO ref_data_sources (slug, name, source_type) VALUES
    ('freedict',  'Free Dictionary API', 'definitions'),
    ('translate', 'Google Translate',    'translations'),
    ('wiktionary','Wiktionary',          'definitions'),
    ('ngsl',      'New General Service List', 'metadata'),
    ('nawl',      'New Academic Word List',   'metadata'),
    ('cmu',       'CMU Pronouncing Dictionary', 'pronunciations'),
    ('wordnet',   'WordNet',             'relations'),
    ('tatoeba',   'Tatoeba',             'examples');

CREATE TABLE ref_entry_source_coverage (
    ref_entry_id    UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    source_slug     TEXT NOT NULL REFERENCES ref_data_sources(slug) ON DELETE CASCADE,
    status          TEXT NOT NULL,
    dataset_version TEXT,
    fetched_at      TIMESTAMPTZ DEFAULT now(),

    PRIMARY KEY (ref_entry_id, source_slug),

    CONSTRAINT chk_ref_entry_source_coverage_status
        CHECK (status IN ('fetched', 'no_data', 'failed'))
);

-- +goose Down
DROP TABLE IF EXISTS ref_entry_source_coverage;
DROP TABLE IF EXISTS ref_data_sources;
