-- +goose Up
CREATE OR REPLACE FUNCTION fn_preserve_sense_on_ref_delete()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE senses s
    SET definition     = COALESCE(s.definition, OLD.definition),
        part_of_speech = COALESCE(s.part_of_speech, OLD.part_of_speech),
        cefr_level     = COALESCE(s.cefr_level, OLD.cefr_level)
    WHERE s.ref_sense_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_preserve_sense_on_ref_delete
    BEFORE DELETE ON ref_senses
    FOR EACH ROW
    EXECUTE FUNCTION fn_preserve_sense_on_ref_delete();

CREATE OR REPLACE FUNCTION fn_preserve_translation_on_ref_delete()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE translations t
    SET text = COALESCE(t.text, OLD.text)
    WHERE t.ref_translation_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_preserve_translation_on_ref_delete
    BEFORE DELETE ON ref_translations
    FOR EACH ROW
    EXECUTE FUNCTION fn_preserve_translation_on_ref_delete();

CREATE OR REPLACE FUNCTION fn_preserve_example_on_ref_delete()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE examples e
    SET sentence    = COALESCE(e.sentence, OLD.sentence),
        translation = COALESCE(e.translation, OLD.translation)
    WHERE e.ref_example_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_preserve_example_on_ref_delete
    BEFORE DELETE ON ref_examples
    FOR EACH ROW
    EXECUTE FUNCTION fn_preserve_example_on_ref_delete();

-- +goose Down
DROP TRIGGER IF EXISTS trg_preserve_example_on_ref_delete ON ref_examples;
DROP FUNCTION IF EXISTS fn_preserve_example_on_ref_delete();

DROP TRIGGER IF EXISTS trg_preserve_translation_on_ref_delete ON ref_translations;
DROP FUNCTION IF EXISTS fn_preserve_translation_on_ref_delete();

DROP TRIGGER IF EXISTS trg_preserve_sense_on_ref_delete ON ref_senses;
DROP FUNCTION IF EXISTS fn_preserve_sense_on_ref_delete();
