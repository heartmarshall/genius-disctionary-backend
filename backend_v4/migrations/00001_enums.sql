-- +goose Up
CREATE TYPE learning_status AS ENUM ('NEW', 'LEARNING', 'REVIEW', 'MASTERED');
CREATE TYPE review_grade    AS ENUM ('AGAIN', 'HARD', 'GOOD', 'EASY');
CREATE TYPE part_of_speech  AS ENUM (
    'NOUN', 'VERB', 'ADJECTIVE', 'ADVERB', 'PRONOUN',
    'PREPOSITION', 'CONJUNCTION', 'INTERJECTION',
    'PHRASE', 'IDIOM', 'OTHER'
);
CREATE TYPE entity_type     AS ENUM ('ENTRY', 'SENSE', 'EXAMPLE', 'IMAGE', 'PRONUNCIATION', 'CARD', 'TOPIC');
CREATE TYPE audit_action    AS ENUM ('CREATE', 'UPDATE', 'DELETE');

-- +goose Down
DROP TYPE IF EXISTS audit_action;
DROP TYPE IF EXISTS entity_type;
DROP TYPE IF EXISTS part_of_speech;
DROP TYPE IF EXISTS review_grade;
DROP TYPE IF EXISTS learning_status;
