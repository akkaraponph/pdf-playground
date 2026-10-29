# Real-world PDF signing API

A runnable service agreement workflow with **Fiber v3**, **gofpdf** generation,
real **digitorus/pdfsign** PAdES B-B signatures, and background goroutine tasks.
It shares the repository's Go module and requires Go 1.27+, Linux or macOS.

## Run

From this directory, the API starts with a generated local test certificate. It binds to localhost by default and document routes have no authentication:

```sh
cd realworld
go run ./cmd/api/main.go
```

You can also run `go run ./realworld/cmd/api/main.go` from the repository root. The defaults are `127.0.0.1:3000` and data in `realworld/data/`. No environment variables, API tokens, or certificate flags are required. The configured `-actor` is recorded for workflow actions. `GET /healthz` is public.

In another terminal, the demo script calls the local API directly. Run the
complete generation → approval → signing → download → verification flow. It
requires `curl` and `jq`:

```sh
bash realworld/scripts/demo.sh
```

Outputs: `realworld/data/demo-output/<document-id>-original.pdf` and
`<document-id>-signed.pdf`. The script polls background status and fails if a
task fails or times out. Set `PDFSIGN_BASE_URL` or `PDFSIGN_OUTPUT_DIR` to override
its server address or output directory.

## API walkthrough

Use a generated service agreement:

```sh
BASE=http://127.0.0.1:3000/api/v1

curl -sS "$BASE/documents/generate" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Service agreement","customer":"Acme Ltd","amount_cents":125050,"currency":"USD"}'
```

The response is `202 Accepted`, with `status: generation_queued`, a document
`id`, a `Location` header for polling, and `Retry-After: 1`. Set `ID` to that ID.
The sample template accepts printable ASCII, positive integer minor units,
and `USD`, `THB` or `EUR`; `125050` means `1250.50`. Unicode requires adding an
appropriate embedded TrueType font to the generator.

Alternatively, upload an existing unsigned PDF as a draft:

```sh
curl -sS "$BASE/documents/upload" \
  -F 'title=Vendor contract' -F 'file=@example1/contract.pdf'
```

Uploads are limited to **10 MiB**, must parse as a PDF with pages, and cannot
already contain a signature. Password-protected input is unsupported.

Poll until `draft`, review the original, then submit and approve:

```sh
ID=<document-id>
curl -sS "$BASE/documents/$ID"
curl -fsS "$BASE/documents/$ID/download?version=original" -o original.pdf

curl -sS -X POST "$BASE/documents/$ID/submit"
curl -sS -X POST "$BASE/documents/$ID/approve" \
  -H 'Content-Type: application/json' -d '{"reason":"Reviewed and approved"}'
```

Approval responds `202 Accepted` with `signing_queued`. Poll until `signed`, then:

```sh
curl -fsS "$BASE/documents/$ID/download" \
  -o signed.pdf
curl -sS "$BASE/documents/$ID/verify"
```

Expected verification response:

```json
{"valid":true,"signatures":1,"trusted":true,"revocation_checked":false,"timestamp_checked":false}
```

`trusted` refers to the explicitly configured CA bundle. The signing certificate
identifies the **server signing identity**. The reviewer approval is a separate
application audit record; this example does not issue personal signing
certificates or claim that the reviewer owns the server's key. B-B signing works
offline, without an RFC 3161 timestamp. Revocation and timestamp trust are not
asserted. Acrobat needs its own explicit trust configuration for the test CA.

## Routes

| Method | Route under `/api/v1` | Result |
| --- | --- | --- |
| POST | `/documents/generate` | Queue gofpdf generation; 202 |
| POST | `/documents/upload` | Multipart `title`, `file`; create draft; 201 |
| GET | `/documents?limit=50&offset=0` | Metadata list; limit 1–100 |
| GET | `/documents/:id` | State, revisions, hashes, events and task error |
| POST | `/documents/:id/submit` | Draft → pending approval |
| POST | `/documents/:id/approve` | JSON `reason`; queue signing; 202 |
| POST | `/documents/:id/reject` | JSON `reason`; reject pending document |
| POST | `/documents/:id/retry` | Retry failed generation/signing; 202 |
| GET | `/documents/:id/download?version=original` | Download unsigned original |
| GET | `/documents/:id/download` | Download committed signed PDF |
| GET | `/documents/:id/verify` | Verify committed signed PDF |

Errors use JSON `error` and `request_id`: 400 for invalid input, 401 for missing
or incorrect credentials, 404 for missing IDs, 409 for unavailable PDF versions,
stale revisions or invalid transitions, and 413 for oversized request bodies.
Submitting, approving or rejecting an already-transitioned document returns 409.
Rejection is final; upload or generate a new document to start another flow.

## Hexagonal structure

```text
realworld/
  cmd/api/main.go                  composition, configuration, shutdown
  internal/document/
    model.go                      domain types, validation, output ports
    service.go                    document use cases
    worker.go                     background task orchestration
  internal/adapter/
    httpapi/api.go                inbound Fiber v3 adapter
    filestore/store.go            outbound atomic JSON snapshot repository
    pdfgenerator/generator.go     outbound gofpdf generator
    pdfsigner/signer.go            outbound pdfsign signer/verifier
  scripts/demo.sh                  executable end-to-end showcase
```

The document package imports no HTTP, PDF or filesystem framework. The
repository, generator and signer interfaces belong to the consuming application.
`main` supplies implementations. This is a small hexagonal service with ordinary
Go errors and complete aggregate snapshots, without an event bus or extra service
layers.

## Background task behavior

Generation: `generation_queued` → `generating` → `draft` or `generation_failed`.
Approval: `draft` → `pending_approval` → `signing_queued` → `signing` → `signed`
or `signing_failed`. A pending document may instead become `rejected`.

The HTTP handler persists the queued state before responding. A single background
worker goroutine scans the durable backlog and is woken through a buffered,
coalescing channel; a one-second ticker prevents missed work. No goroutine is
created per request, and task data lives on disk rather than in the channel.
Each PDF library gets a fresh document instance per task. Only verified signed
bytes are committed. Failed tasks preserve the original and record an error;
use `/retry` after resolving the cause.

On startup, `generating` and `signing` records are requeued. If a process dies
after computing a PDF but before saving it, that computation may run again;
the visible result is committed once using revision checks. Graceful shutdown
drains HTTP, cancels and joins the worker, then closes storage. A task interrupted
by graceful cancellation is recorded as failed and can be retried. A PDF task has
a two-minute cooperative context deadline; gofpdf's CPU work itself does not offer
preemptive cancellation.

## Persistence and showcase limits

Metadata, audit events, original and signed bytes are saved together in one JSON
snapshot per document (PDF bytes are base64 encoded). Writes use a private temp
file, file sync, atomic rename, then directory sync. Revisions prevent competing decisions from
overwriting each other. An OS file lock enforces one owner per data directory.
This provides local restart recovery on a filesystem honoring those sync calls;
it is not a distributed queue. A directory-sync failure after rename is an
uncertain commit: reload the document before retrying. Do not use a network filesystem.

The repository reads complete snapshots when listing and scanning, so it is
intended for a small showcase dataset. The persistent backlog and retained files
have no quota. For a deployed service, replace the repository with database
transactions and a job queue, keep PDFs in object storage, and use managed keys,
per-user authentication/authorization, TLS, retention and immutable audit storage.
The document API has no authentication or authorization and grants every caller
with network access all document operations. Keep it bound to localhost; do not
expose it to a shared network or deploy it as-is. Runtime data and generated
credentials are ignored by Git.

## Verify the implementation

From the repository root:

```sh
go test ./...
go test -race ./realworld/...
go vet ./...
go build -o /tmp/pdfsign-api ./realworld/cmd/api
```

Tests use fresh local credentials, real gofpdf and pdfsign, and temporary storage.
They cover asynchronous lifecycle behavior, signature integrity, tampering,
unauthenticated local API access, input validation, rejection, revision
conflicts, exclusive store ownership, restart persistence, interrupted-task
recovery and failure retry.
No public timestamp authority or other external service is needed.


## React web interface

The React workflow UI lives in [web-ui/](web-ui/). Run `pnpm install` and
`pnpm run dev` there; it connects to this API through a local Vite proxy.
