-- +goose Up
ALTER TABLE glossary_terms ADD COLUMN category TEXT;
ALTER TABLE glossary_terms ADD COLUMN avoid TEXT;

-- +goose Down
ALTER TABLE glossary_terms DROP COLUMN avoid;
ALTER TABLE glossary_terms DROP COLUMN category;
