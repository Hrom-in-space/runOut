BEGIN;

CREATE TABLE needs (
    id   BIGSERIAL PRIMARY KEY,
    name text      NOT NULL
);

COMMIT;
