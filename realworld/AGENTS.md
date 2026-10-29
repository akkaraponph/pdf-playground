# Instructions for `realworld/`

This directory contains a runnable local PDF approval and signing API. Read
[`README.md`](README.md) for setup, routes, lifecycle details, and operational
limits before changing behavior.

## Structure and boundaries

- `internal/document` owns document states, validation, use cases, and the
  repository, generator, and signer interfaces. Keep it independent of HTTP,
  PDF libraries, and filesystem details.
- `internal/adapter` implements those ports. Keep transport and library-specific
  behavior in the relevant adapter.
- `cmd/api` wires adapters, configuration, the worker, and graceful shutdown.
- `scripts/demo.sh` is the end-to-end local walkthrough.
- `docs/knowledge/` is the home for durable, reusable technical knowledge
  discovered while working on this service.
- The Go module is at the repository root. Run Go commands from there and use
  `./realworld/...` package paths.

## Knowledge documentation

- When implementation work uncovers a non-obvious design decision, operational
  constraint, integration detail, or failure mode that will help future work,
  write or update a focused Markdown note under `docs/knowledge/` as part of
  that work.
- Keep knowledge notes practical and specific to this service. Explain the
  context, the applicable guidance, and any verification or references needed
  to use the information safely.
- Check `docs/knowledge/` before creating a note. Extend an existing note when
  it covers the same topic; avoid duplicate or speculative documentation.
- Keep endpoint usage and the supported walkthrough in `README.md`, lifecycle
  and operational behavior close to the relevant code or README, and reusable
  implementation learnings in `docs/knowledge/`.
- Update or remove a knowledge note when code changes make it inaccurate. Do
  not present generated or inferred details as verified facts; label
  assumptions and link to the source of truth where practical.

## Safety and behavior

- Preserve the state machine and revision checks. Persist queued state before
  returning an accepted response, and keep task data durable rather than relying
  on in-memory queue messages.
- Commit signed PDF bytes only after verification. Preserve the original on
  signing or persistence failures, and keep retries explicit.
- Treat reviewer approval and the configured server signing certificate as
  separate identities. Do not claim the reviewer owns the signing key.
- This showcase API intentionally has no authentication; keep its default bind
  address on localhost. Never accept an actor identity from request JSON or log
  private key material or PDF contents.
- Do not commit generated credentials, runtime data, or demo output. `certs/`
  contains local test material; keep private keys out of source control.
- This is a local showcase, not a distributed queue or compliance ledger. Do
  not describe its JSON event history as tamper-proof or infer revocation or
  timestamp trust unless those checks are actually implemented.
- Keep test and demo data in temporary or ignored directories. Avoid using the
  checked-in workspace data as a test fixture that a command may overwrite.

## Verification

When changing code, run the relevant Go tests from the repository root; for
cross-cutting changes, use the verification commands listed in the README:

```sh
go test ./realworld/...
go test -race ./realworld/...
go vet ./...
```

The demo requires explicit credentials and a local server; follow the README
instead of starting it implicitly.
