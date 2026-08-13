-- lesson_content.content_type ya acepta 'markdown', pero no había ninguna
-- columna para guardar ese texto — youtube_url y pdf_url alcanzan para los
-- otros dos tipos, pero el markdown en sí no tenía dónde vivir.
ALTER TABLE lesson_content ADD COLUMN body TEXT;
