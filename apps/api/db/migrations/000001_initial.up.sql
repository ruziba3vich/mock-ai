CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE identity_provider AS ENUM ('telegram', 'email', 'google');
CREATE TYPE user_role AS ENUM ('student', 'admin');
CREATE TYPE exam_skill AS ENUM ('listening', 'reading');
CREATE TYPE exam_status AS ENUM ('draft', 'published', 'archived');
CREATE TYPE question_type AS ENUM (
    'multiple_choice',
    'multiple_select',
    'true_false_not_given',
    'yes_no_not_given',
    'short_answer',
    'sentence_completion',
    'summary_completion',
    'matching',
    'map_labeling'
);
CREATE TYPE attempt_status AS ENUM ('in_progress', 'submitted', 'expired', 'scored');

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name text NOT NULL,
    avatar_url text,
    email text,
    email_verified_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX users_email_unique
    ON users (lower(email))
    WHERE email IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE user_identities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider identity_provider NOT NULL,
    provider_subject text NOT NULL,
    provider_email text,
    profile jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_subject)
);

CREATE TABLE password_credentials (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_hash text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE user_role_assignments (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role user_role NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role)
);

CREATE TABLE sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    user_agent text,
    ip_address inet,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz
);

CREATE INDEX sessions_active_user_idx ON sessions (user_id, expires_at)
    WHERE revoked_at IS NULL;

CREATE TABLE exams (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug text NOT NULL UNIQUE,
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    archived_at timestamptz
);

CREATE TABLE exam_versions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id uuid NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    version integer NOT NULL CHECK (version > 0),
    title text NOT NULL,
    description text NOT NULL DEFAULT '',
    skill exam_skill NOT NULL,
    status exam_status NOT NULL DEFAULT 'draft',
    duration_seconds integer NOT NULL CHECK (duration_seconds > 0),
    instructions text NOT NULL DEFAULT '',
    published_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (exam_id, version),
    CHECK ((status = 'published' AND published_at IS NOT NULL) OR status <> 'published')
);

CREATE UNIQUE INDEX one_published_version_per_exam
    ON exam_versions (exam_id)
    WHERE status = 'published';

CREATE TABLE exam_sections (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_version_id uuid NOT NULL REFERENCES exam_versions(id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position > 0),
    title text NOT NULL,
    instructions text NOT NULL DEFAULT '',
    audio_url text,
    transcript text,
    passage text,
    UNIQUE (exam_version_id, position)
);

CREATE TABLE questions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    section_id uuid NOT NULL REFERENCES exam_sections(id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position > 0),
    type question_type NOT NULL,
    prompt text NOT NULL,
    points numeric(6,2) NOT NULL DEFAULT 1 CHECK (points > 0),
    explanation text NOT NULL DEFAULT '',
    config jsonb NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (section_id, position)
);

CREATE TABLE question_options (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id uuid NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position > 0),
    label text NOT NULL,
    value text NOT NULL,
    UNIQUE (question_id, position)
);

CREATE TABLE accepted_answers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id uuid NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    value text NOT NULL,
    is_case_sensitive boolean NOT NULL DEFAULT false,
    position integer NOT NULL DEFAULT 1 CHECK (position > 0),
    UNIQUE (question_id, value)
);

CREATE TABLE band_conversions (
    exam_version_id uuid NOT NULL REFERENCES exam_versions(id) ON DELETE CASCADE,
    minimum_raw_score numeric(7,2) NOT NULL CHECK (minimum_raw_score >= 0),
    band numeric(2,1) NOT NULL CHECK (band >= 0 AND band <= 9),
    PRIMARY KEY (exam_version_id, minimum_raw_score),
    UNIQUE (exam_version_id, band)
);

CREATE TABLE attempts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id),
    exam_version_id uuid NOT NULL REFERENCES exam_versions(id),
    status attempt_status NOT NULL DEFAULT 'in_progress',
    started_at timestamptz NOT NULL DEFAULT now(),
    deadline_at timestamptz NOT NULL,
    submitted_at timestamptz,
    scored_at timestamptz,
    raw_score numeric(7,2),
    band_score numeric(2,1),
    question_count integer NOT NULL CHECK (question_count > 0),
    CHECK (deadline_at > started_at),
    CHECK ((status IN ('submitted', 'expired', 'scored') AND submitted_at IS NOT NULL) OR status = 'in_progress'),
    CHECK ((status = 'scored' AND scored_at IS NOT NULL AND raw_score IS NOT NULL AND band_score IS NOT NULL) OR status <> 'scored')
);

CREATE INDEX attempts_user_history_idx ON attempts (user_id, started_at DESC);
CREATE UNIQUE INDEX one_active_attempt_per_exam
    ON attempts (user_id, exam_version_id)
    WHERE status = 'in_progress';

CREATE TABLE attempt_answers (
    attempt_id uuid NOT NULL REFERENCES attempts(id) ON DELETE CASCADE,
    question_id uuid NOT NULL REFERENCES questions(id),
    answer jsonb NOT NULL,
    is_correct boolean,
    awarded_points numeric(6,2),
    revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
    saved_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (attempt_id, question_id)
);

CREATE TABLE attempt_analyses (
    attempt_id uuid NOT NULL REFERENCES attempts(id) ON DELETE CASCADE,
    dimension text NOT NULL,
    dimension_value text NOT NULL,
    correct_count integer NOT NULL CHECK (correct_count >= 0),
    question_count integer NOT NULL CHECK (question_count > 0),
    score numeric(7,2) NOT NULL CHECK (score >= 0),
    PRIMARY KEY (attempt_id, dimension, dimension_value)
);
