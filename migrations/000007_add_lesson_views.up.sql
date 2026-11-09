-- "Lección vista" vive en la misma fila de lesson_progress que "completada":
-- una fila con completed = false y first_viewed_at cargado significa que el
-- usuario abrió la lección pero todavía no la marcó como completa. Las
-- queries de progreso existentes ya filtran por completed, así que estas
-- filas nuevas no cambian ningún conteo.
ALTER TABLE lesson_progress
    ADD COLUMN first_viewed_at TIMESTAMPTZ NULL,
    ADD COLUMN last_viewed_at  TIMESTAMPTZ NULL;

-- Toda lección completada antes de esta migración se asume vista en el
-- momento en que se completó.
UPDATE lesson_progress
SET first_viewed_at = completed_at,
    last_viewed_at  = completed_at
WHERE completed AND completed_at IS NOT NULL;
