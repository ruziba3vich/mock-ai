# Speaking interviews

New Speaking sessions are real-time voice conversations, not an official IELTS result. Two original topic sets guide 6 introductory turns, a 60-second preparation / 120-second long turn, and 4 discussion turns. The examiner can adapt follow-up questions while preserving this structure. The server assigns a regional voice at creation and preserves it on reconnect. Selection metadata is not displayed to students. Existing recorded sessions remain supported.

## Configuration and deployment

Migration `000011_speaking` adds sessions, turns, and deletion-resistant daily usage records. `000012_live_speaking` adds the session mode, pinned Live model, connection leases/counters and live transcripts. All SQL is managed through sqlc. CI runs migrations through the existing migration executable. No changes to any other database or published database port are needed.

- `AI_KEY_ENCRYPTION_KEY`: existing 32-byte hex secret; never commit it or change it without re-encrypting stored keys AND recordings.
- `GEMINI_SPEAKING_MODEL`: default `gemini-3.7-flash`, evaluates actual audio.
- `GEMINI_SPEAKING_VOICE_MODEL`: default `gemini-3.8-flash-tts`, uses Interactions API and verified regional voice IDs.
- `GEMINI_SPEAKING_LIVE_MODEL`: default `gemini-3.8-live`, verified against the actual Live WebSocket API with the complete tool/VAD setup.
- Reuses the encrypted Gemini key vault from Writing. GitHub Actions variables set the models; the existing Actions secret supplies the encryption key. No browser API key.

Live audio requires HTTPS (localhost permitted), AudioContext/AudioWorklet, WebSocket and microphone permission. Telegram webview support depends on its client version. Use headphones to avoid examiner playback reaching the microphone. Microphone forwarding pauses during examiner playback and Part 2 preparation; students currently cannot interrupt examiner playback by speaking over it.

## Live transport and turn-taking

`POST /api/v1/speaking/live/sessions` with `{"consent":true}` creates a live session. `GET /api/v1/speaking/live/sessions/{id}/stream` upgrades to WebSocket, checking both the student cookie and exact Origin. The backend proxies Gemini; keys never reach the browser. Input binary frames contain PCM16 little-endian mono 16 kHz, normally 100 ms/3200 bytes. Output binary frames are PCM16 mono 24 kHz. JSON events: `state`, `audio_end`, `interrupted`, `heartbeat`, `done`. Client controls: `playback_done`, `repeat`, `done_speaking`.

Native VAD uses low speech-start/end sensitivity and 3500 ms silence duration. That silence is an opportunity for semantic judgment, not permission to advance. The model must call `wait_for_student` for incomplete thoughts/thinking/noise, or `advance_question` after a complete substantive answer. Server output gating suppresses unapproved speech; progression requires meaningful transcription plus valid audio and is saved before another question. Thinking pauses remain within the same answer. Model-based turn detection is probabilistic, not a guarantee about intent. At the two-minute answer cap, the controller moves on only if a valid answer exists. Part 2 preparation is timed on the server.

One active connection is leased for 45 seconds, renewed every 15 seconds. Stale connections cannot save or release a newer lease. Sessions allow at most 12 successful connections, each capped at 20 minutes. Input and recordings are bounded; input throughput is rate-limited. Disconnects, provider GoAway, backgrounding and manual pause preserve completed answers, but require repeating the unfinished answer with the same examiner. There is no claim of seamless mid-answer recovery. The final answer automatically queues the existing audio-based report.

## HTTP contract

All routes require the student session cookie; mutations require an exact student-app Origin. Responses and audio use private/no-store caching. Audio access always checks ownership and expiry.

| Method | `/api/v1/speaking` suffix | Result |
|---|---|---|
| GET | `/settings` | enabled, countries, daily limit, retention days |
| GET | `/sessions` | latest 30 sessions |
| POST | `/sessions` | `{ "consent": true }`; assigns examiner, queues first question |
| GET / DELETE | `/sessions/{id}` | interview detail / delete interview and all audio |
| GET | `/sessions/{id}/turns/{position}/examiner` | generated question WAV |
| GET | `/sessions/{id}/turns/{position}/recording` | decrypted student WAV |
| PUT | `/sessions/{id}/turns/{position}/answer` | raw canonical WAV, saves current answer and queues next voice |
| POST | `/sessions/{id}/turns/{position}/skip` | explicitly skips current answer |
| POST | `/sessions/{id}/turns/{position}/voice/retry` | retries failed current question, max 3 total attempts |
| POST | `/sessions/{id}/submit` | queues report after all 11 turns are answered/skipped |

Student WAV format: 44-byte PCM header, 16 kHz mono, signed 16-bit little endian, 1 second minimum, per-turn maximum enforced server-side (2 seconds transport tolerance), 4.2 MB hard body limit. The browser transcodes its native recording; the server checks format, length, duration and non-silent signal. A signal check does not guarantee speech; the evaluator must withhold scores for insufficient intelligible speech.

## Persistence, privacy and failure recovery

Recordings are AES-256-GCM encrypted in PostgreSQL, with authenticated data binding each ciphertext to its Speaking purpose, session and question. Provider-key ciphertext cannot be decoded as a recording. Recordings and generated question audio expire after 30 days; cleanup runs even if evaluation is disabled. Reports/transcripts remain until student deletion. Users consent before their recordings are sent to Google. Larger combined recordings use Gemini Files and request deletion after evaluation; provider-side retention/data policies still apply. Do not claim zero retention.

One open interview per student, three new interviews/day and five evaluation requests/day (UTC). Deleting an interview does not reset its usage. Durable DB queues use leases, run IDs and guarded completion; retries cannot overwrite a newer run, and deletion cannot be undone by an in-flight worker. Failed/interrupted reports are retryable. Saved answers resume after refresh; unsaved recordings cannot be recovered after leaving. Session deletion cascades to turns.

The live migration refuses downgrade while live interviews exist, preventing silent loss of their transport/progression metadata. Roll application code forward to recover a failed rollout; do not delete student interviews to force a downgrade.

The report includes four criterion estimates (fluency/coherence, vocabulary, grammar, pronunciation), overall Speaking estimate computed server-side, transcripts, strengths, answer development feedback, timestamped issues and practice steps. Pronunciation is assessed from sound and intelligibility, never inferred solely from a transcript or penalized for a non-native accent. Invalid quotes, timestamps, criterion keys and band values reject the report. Under 60 seconds of audio, fewer than three answers, or missing any part suppresses all band estimates. No four-skill overall IELTS band is invented.

## Verification

`go test ./...` covers waveform/report validation and provider contracts. `MOCKAI_INTEGRATION=1 DATABASE_URL=... go test ./...` requires a fresh disposable `mock_ai_learning_test` database; it tests authenticated HTTP, ownership, CSRF, encryption/decryption, ordering, worker execution, computed scores, expiry, usage limits, and migration rollback/reapply. Integration scoring uses a deterministic provider double; it does not validate AI assessment accuracy.

Frontend: `npm test`, `npm run build`, `npm run test:e2e`. Normal Playwright checks mobile Speaking discovery/consent with the real backend. Opt-in `SPEAKING_LIVE_E2E=1 SPEAKING_LIVE_FIXTURES=/absolute/fixture-directory SPEAKING_THINKING_FIXTURE=/absolute/thinking.wav npm run test:e2e -- e2e/speaking.spec.ts` needs an isolated backend configured with Gemini. Fixtures are eleven distinct 16 kHz WAV answers (`0.wav` to `10.wav`) plus an unfinished thought asking for time to think. Synthesized speech goes through Chromium's real AudioWorklet, WebSocket and provider. The test checks silence/thinking pauses, saves all 11 answers, reconnects, obtains a real report, plays saved audio and deletes the interview. This incurs provider usage and validates transport/behavior, not human-accent scoring accuracy. Physical Telegram iOS/Android microphone checks and human/examiner calibration remain separate release checks.
