# sample-project

SDMS Teacher Management — a Go service exposing teacher record management
(create, read, update, deactivate) plus owned education-history rows. Design
stage: the repository contains no application code yet. What exists is the
confirmed design for FEAT-001, under `docs/FEAT-001/`.

## Stack

**Go** on **`github.com/nexsoft-git/nexcommon`**, backed by **PostgreSQL**.
Decided during the FEAT-001 architecture pass — recorded here with its reasons
so a session starting cold doesn't re-litigate it blind:

- **Go** — the pipeline downstream of the design is `/go-dev`, and the
  environment's generators emit Go and Postgres DDL. Nothing in the feature
  argued for anything else.
- **`nexcommon`** — the team's shared backend library. It supplies the HTTP
  controller wrapping, the validator strategies, the DAO get-list flow, audit
  logging and the regex constants. Reusing it is not a preference: the spec's
  "the project's phone-number regex" turned out to mean a constant in this
  library, not something any service owns.
- **PostgreSQL** — the feature is plain transactional relational CRUD, and
  `nexcommon`'s DAO layer targets it directly. The concurrency requirement is
  met by a row-level conditional `UPDATE`, which is what Postgres does well.

## Conventions

Depends on `github.com/nexsoft-git/nexcommon` — follow `nexcommon-go-standards`,
`nexcommon-go-config-standards`, `nexcommon-go-data-standards` and
`nexcommon-go-project-layout`; `golang-testing` for tests. Read those skills
rather than a restatement of them here.

Layout follows `nexcommon-go-project-layout`'s default tree — no deviations
were decided.

## Feature docs

| Feature | Summary | Docs |
| :--- | :--- | :--- |
| FEAT-001 | Teacher CRUD: create, read, update and deactivate teacher records, including their education history. | [docs/FEAT-001/](docs/FEAT-001/) |
