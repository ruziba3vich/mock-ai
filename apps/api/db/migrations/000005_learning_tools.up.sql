CREATE TABLE attempt_workspace (
    attempt_id uuid PRIMARY KEY REFERENCES attempts(id) ON DELETE CASCADE,
    data jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE mistake_retries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question_id uuid NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    answer jsonb NOT NULL,
    is_correct boolean NOT NULL,
    attempted_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX mistake_retries_user_question ON mistake_retries(user_id, question_id, attempted_at DESC);
CREATE TABLE study_reminders (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    enabled boolean NOT NULL DEFAULT false,
    local_time text NOT NULL DEFAULT '19:00' CHECK (local_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    timezone text NOT NULL DEFAULT 'Asia/Tashkent',
    last_sent_date date,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE content_drafts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id uuid REFERENCES exams(id),
    author_id uuid NOT NULL REFERENCES users(id),
    data jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);
