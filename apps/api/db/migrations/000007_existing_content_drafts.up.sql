-- Make legacy published material editable without modifying its live version.
INSERT INTO content_drafts (id,exam_id,author_id,data)
SELECT e.id,e.id,e.created_by,jsonb_build_object(
 'title',v.title,'description',v.description,'skill',v.skill,'purpose',v.purpose,
 'duration_seconds',v.duration_seconds,'instructions',v.instructions,
 'bands',COALESCE((SELECT jsonb_agg(jsonb_build_object('minimum',b.minimum_raw_score::integer,'band',b.band) ORDER BY b.minimum_raw_score) FROM band_conversions b WHERE b.exam_version_id=v.id),'[]'::jsonb),
 'sections',COALESCE((SELECT jsonb_agg(jsonb_build_object(
  'title',s.title,'instructions',s.instructions,'passage',COALESCE(s.passage,s.transcript,''),'audio_url',COALESCE(s.audio_url,''),
  'questions',COALESCE((SELECT jsonb_agg(jsonb_build_object('type',q.type,'prompt',q.prompt,'explanation',q.explanation,
   'options',COALESCE((SELECT jsonb_agg(o.value ORDER BY o.position) FROM question_options o WHERE o.question_id=q.id),'[]'::jsonb),
   'answers',COALESCE((SELECT jsonb_agg(a.value ORDER BY a.position) FROM accepted_answers a WHERE a.question_id=q.id),'[]'::jsonb)
  ) ORDER BY q.position) FROM questions q WHERE q.section_id=s.id),'[]'::jsonb)
 ) ORDER BY s.position) FROM exam_sections s WHERE s.exam_version_id=v.id),'[]'::jsonb)
)
FROM exams e JOIN exam_versions v ON v.exam_id=e.id
WHERE v.status='published' AND NOT EXISTS(SELECT 1 FROM content_drafts d WHERE d.exam_id=e.id);
