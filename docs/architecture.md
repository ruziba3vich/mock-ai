# Architecture

The backend is a Go modular monolith. Modules communicate through application-level interfaces and share PostgreSQL. SQL is explicit and compiled by sqlc.

| Module | Owns |
| --- | --- |
| Identity | users, external identities, credentials, sessions, roles |
| Exam | exam versions, sections, content, questions, answer keys, publication |
| Attempt | timed sessions, answer snapshots, submission lifecycle |
| Scoring | correctness rules, raw score, band conversion, analyses |
| Progress | student history and read-optimized statistics |

Domain packages contain business rules and have no HTTP or database dependencies. Application services coordinate use cases. PostgreSQL adapters implement repository interfaces using sqlc-generated code. Transport packages validate requests and translate errors.

The API and web application should be served under one site in production so secure, HTTP-only session cookies work consistently in normal browsers and Telegram WebViews.

