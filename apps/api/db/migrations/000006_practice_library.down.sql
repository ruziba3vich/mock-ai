DELETE FROM content_drafts WHERE exam_id IN (SELECT id FROM exams WHERE slug IN ('practice-library-of-things','practice-cooling-schoolyard','practice-museum-workshop','practice-garden-orientation'));
DELETE FROM exams WHERE slug IN ('practice-library-of-things','practice-cooling-schoolyard','practice-museum-workshop','practice-garden-orientation');
