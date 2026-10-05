DELETE FROM exams WHERE slug IN ('practice-reading-city-bees', 'practice-listening-train-station');
DROP INDEX IF EXISTS exam_versions_published_purpose_idx;
ALTER TABLE exam_versions DROP COLUMN purpose;
DROP TYPE exam_purpose;
