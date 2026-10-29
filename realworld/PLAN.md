# PDF signing API implementation plan

Goal: demonstrate a real contract approval and PDF signing workflow using Fiber v3 and hexagonal boundaries.

Architecture: the document package owns lifecycle rules and repository/signer ports; HTTP, filesystem, and pdfsign adapters depend inward. The existing root Go module is reused. Local JSON snapshots contain metadata, PDF bytes and audit history, committed atomically with optimistic revision checks. One process owns a data directory.

Scope: generate a contract using github.com/phpdave11/gofpdf or upload an unsigned PDF, submit for approval, approve and digitally sign or reject, download original/signed bytes, and verify integrity with an explicitly configured CA. Generation and signing run in a background goroutine; requests return 202 and clients poll document status. Persist queued/running states, recover interrupted work on startup, and expose explicit retry for failed tasks. A bounded number of workers consume a durable local backlog. The unauthenticated API is intended for localhost-only showcase use. Reviewer identity is configured server-side, separate from the certificate identity. Signing uses PAdES B-B offline. Credentials must be supplied explicitly and are never committed.

- [x] Write lifecycle, persistence, real signing and HTTP integration tests; run them before implementation.
- [x] Implement document model, state validation, repository and signer ports and use cases.
- [x] Implement atomic local storage and real pdfsign signing/verification, including key/chain validation.
- [x] Implement Fiber v3 routes, limits, error mapping and graceful shutdown.
- [x] Document setup, full curl walkthrough, architecture and operational limits; run test, race, vet, and build checks.

Review focus: malformed PDF, invalid state transition, mismatched credentials, simultaneous decisions, failed signing/write preserving pending state, restart persistence, unauthorized access and signed output integrity.

Decisions: retain the user-requested directory in the existing checkout; avoid moving unrelated uncommitted work. No database or external TSA is needed for this local showcase. Local audit records are diagnostic and are not a tamper-proof compliance ledger.

Verification: repository-wide tests and vet passed; realworld race tests passed; API binary built. A live HTTP run of scripts/demo.sh generated and signed a contract, returned successful verification and shut down gracefully. Poppler pdfsig independently reported a valid signature over the entire document, with the expected untrusted test CA warning (and a local NSS initialization warning). The credential policy check passed. Code review found a missing directory sync after rename; fixed before final verification.
