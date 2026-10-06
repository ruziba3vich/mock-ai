CREATE TABLE community_profiles (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    nickname text NOT NULL CHECK (char_length(nickname) BETWEEN 2 AND 24),
    enabled boolean NOT NULL DEFAULT false
);

-- Only persisted study activity counts. Empty submissions earn no credit.
CREATE VIEW community_activity AS
SELECT user_id, 'p:' || question_id::text AS activity_key, attempted_at AS occurred_at, 1 AS points
FROM practice_answers
UNION ALL
SELECT a.user_id, 'm:' || a.exam_version_id::text, a.scored_at, 10
FROM attempts a
WHERE a.status = 'scored' AND EXISTS (
    SELECT 1 FROM attempt_answers aa WHERE aa.attempt_id = a.id
      AND jsonb_array_length(coalesce(aa.answer->'values', '[]'::jsonb)) > 0
);

CREATE VIEW community_weekly_scores AS
SELECT p.user_id, p.nickname, sum(c.points)::integer AS points
FROM community_profiles p
JOIN (SELECT DISTINCT user_id, activity_key, points FROM community_activity
      WHERE occurred_at >= date_trunc('week',now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC') c ON c.user_id=p.user_id
WHERE p.enabled
GROUP BY p.user_id,p.nickname;

CREATE VIEW community_weekly_ranks AS
SELECT user_id,nickname,points,rank() OVER (ORDER BY points DESC)::integer AS position,
row_number() OVER (ORDER BY points DESC,user_id) AS ordinal FROM community_weekly_scores;
