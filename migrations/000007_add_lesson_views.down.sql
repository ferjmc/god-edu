-- Las filas de "solo vista" no tienen sentido sin estas columnas.
DELETE FROM lesson_progress WHERE NOT completed;

ALTER TABLE lesson_progress
    DROP COLUMN first_viewed_at,
    DROP COLUMN last_viewed_at;
