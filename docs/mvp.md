# Listening and Reading MVP

## Product boundary

The same responsive client runs in a Telegram Mini App and in a normal desktop browser. Students enter through Telegram, email/password, or Google. Administrators use the browser application.

### Student journey

1. Sign in and land on the student dashboard.
2. Browse available Listening and Reading mock tests.
3. Start a timed attempt and receive a stable question order.
4. Save answers continuously and resume an unfinished attempt.
5. Submit manually or automatically when time expires.
6. Receive raw score, IELTS band score, answer review, and skill/question-type breakdown.
7. Review attempt history and progress over time.

### Admin journey

1. Create a draft exam and its Listening or Reading sections.
2. Add passages, audio, questions, accepted answers, and scoring metadata.
3. Preview, publish, unpublish, and archive exams.
4. Inspect students, attempts, completion rates, and score distributions.

## Rules for the first release

- Only Listening and Reading exams can be published.
- Published exam content is immutable for attempts already started. Editing creates a new exam version.
- The server owns timers and submission state. A client clock is only presentational.
- Answers are autosaved and each save is idempotent.
- Telegram identity is accepted only after validating Mini App `initData` with the bot token.
- Email passwords are stored as Argon2id hashes.
- Google login uses Authorization Code flow with PKCE.
- A person may link Telegram, email, and Google identities to one user.
- Raw-to-band conversion is stored per exam version because IELTS conversion can vary by test.

## Deferred

- Writing and Speaking exam delivery and human review
- AI teacher, generated feedback, chat, recommendations, and adaptive study plans
- Payments, subscriptions, organizations, and live proctoring

