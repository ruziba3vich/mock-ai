-- Remove only imported editor drafts; published versions remain intact.
DELETE FROM content_drafts WHERE id=exam_id;
