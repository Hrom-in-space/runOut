-- name: ListNeeds :many
SELECT name
FROM needs
ORDER BY name;

-- name: CreateNeed :exec
INSERT INTO needs (
    name
) VALUES ($1);
