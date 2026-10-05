CREATE TYPE ielts_track AS ENUM ('academic', 'general_training');
CREATE TYPE study_event_type AS ENUM ('attempt_review');

CREATE TABLE student_profiles (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    target_band numeric(2,1) NOT NULL DEFAULT 7.0,
    exam_date date,
    track ielts_track NOT NULL DEFAULT 'academic',
    daily_goal_minutes integer NOT NULL DEFAULT 20,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (target_band >= 4.0 AND target_band <= 9.0),
    CHECK (target_band * 2 = trunc(target_band * 2)),
    CHECK (daily_goal_minutes BETWEEN 10 AND 180)
);

CREATE TABLE practice_answers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question_id uuid NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    answer jsonb NOT NULL,
    is_correct boolean NOT NULL,
    attempted_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX practice_answers_user_activity_idx
    ON practice_answers (user_id, attempted_at DESC);

CREATE INDEX practice_answers_user_question_idx
    ON practice_answers (user_id, question_id, attempted_at DESC);

CREATE TABLE study_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type study_event_type NOT NULL,
    attempt_id uuid REFERENCES attempts(id) ON DELETE CASCADE,
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX study_events_user_activity_idx
    ON study_events (user_id, occurred_at DESC);
