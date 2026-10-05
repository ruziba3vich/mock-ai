# HTTP API

All request and response bodies use JSON. Authentication uses the `mock_ai_session` secure, HTTP-only cookie. Except for registration and login, application endpoints require that cookie.

## Identity

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/v1/auth/register` | Create an email/password student and session |
| `POST` | `/api/v1/auth/login` | Authenticate an email/password student |
| `POST` | `/api/v1/auth/telegram` | Verify Telegram Mini App `initData` and authenticate |
| `GET` | `/api/v1/auth/google/start` | Start Google Authorization Code + PKCE flow |
| `GET` | `/api/v1/auth/google/callback` | Verify Google callback and create a session |
| `POST` | `/api/v1/auth/logout` | Revoke the current session |
| `GET` | `/api/v1/me` | Return the current user and roles |

Google login automatically links to an existing account when Google returns the same verified email. Telegram uses the stable Telegram user ID as its identity subject.

## Exams and attempts

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/exams` | List published Listening and Reading exams |
| `GET` | `/api/v1/exams/{exam_id}` | Load published sections, material, questions, and options |
| `POST` | `/api/v1/exams/{exam_id}/attempts` | Start an attempt or resume its active attempt |
| `GET` | `/api/v1/attempts/{attempt_id}` | Restore an owned attempt and its saved answers |
| `PUT` | `/api/v1/attempts/{attempt_id}/answers/{question_id}` | Save an owned answer while the server timer is active |
| `POST` | `/api/v1/attempts/{attempt_id}/submit` | Lock and score an attempt transactionally |
| `GET` | `/api/v1/me/attempts` | Return paginated attempt history |
| `GET` | `/api/v1/me/progress` | Aggregate score, band, and accuracy statistics per skill |
| `GET` | `/api/v1/attempts/{attempt_id}/review` | Return owned scored answers, accepted answers, and explanations |
| `POST` | `/api/v1/attempts/{attempt_id}/reviewed` | Record an owned scored-attempt review as study activity |

## Focused practice

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/practice/questions?skill=reading&type=multiple_choice&limit=5` | Build an untimed set from dedicated practice content |
| `POST` | `/api/v1/practice/questions/{question_id}/check` | Check one practice answer and reveal its accepted answer and explanation |

Practice checks accept the same `{"values":["answer"]}` shape as mock answers. Practice content is separated from scored mock content at the database level, so checking an answer cannot expose a mock-exam answer key. Practice checks are formative and do not affect band history.

## Study planning

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/me/study-plan` | Return study profile, streak, daily activity, seven-day activity, and a weak-area recommendation |
| `PUT` | `/api/v1/me/study-profile` | Update target band, exam date, IELTS track, and daily minute goal |

Study profile updates accept `target_band` in half-band steps from 4.0 to 9.0, an optional future `exam_date` in `YYYY-MM-DD` form, `track` as `academic` or `general_training`, and `daily_goal_minutes` from 10 to 180. General Training content is reserved by the API but is not yet exposed in the student UI.

Answer saves accept `{"values":["answer"]}`. Multiple-select questions use more than one value. Each successful overwrite increments the saved answer revision.
