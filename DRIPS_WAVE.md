# Stellar Wave — Application Package

Everything `AstroForgeLabs/stellar-sentinel` needs to be accepted as a
maintainer repo in the Stellar Wave Program on Drips, and to run it well.

---

## 1. Application checklist (human steps only you can do)

- [ ] **Repo is Public** — confirm in GitHub → Settings. (`stellar-sentinel` is already public.)
- [ ] **Set your Ethereum address** in [`FUNDING.json`](./FUNDING.json) (already set).
- [ ] **Claim the project on Drips** — [drips.network](https://drips.network) → connect wallet → Projects → **Claim**. Drips reads `FUNDING.json` and verifies the wallet you connect owns that address.
- [ ] **Install the Drips Wave GitHub App** on the `AstroForgeLabs` org (read/write on issues, labels, PRs). Only needed once per org.
- [ ] **Create the complexity labels** (see §3).
- [ ] **File the six issues** from §4 and apply one `complexity:*` label + `drips-wave` to each.
- [ ] **Apply to the Stellar Wave Program** on the Drips Wave app: **Maintainers → Orgs and Repos** → sync the repo → **Apply** to the Stellar program, and wait for approval.
- [ ] **Complete KYC / identity verification** on Drips (required before rewards can be distributed).

> If rejected, appeal from **Maintainers → Orgs and Repos** (first appeal 2 weeks
> after rejection; max 3 appeals; must show substantive change).

---

## 2. FUNDING.json

`FUNDING.json` at the repo root declares the wallet that owns the project:

```json
{ "drips": { "ethereum": { "ownedBy": "0x468ed4Da3a1cbe2bCfdc0366Ab035E6EDAd666eD" } } }
```

Drips uses this to verify you are the legitimate maintainer when you claim the
project. If a future Wave settles on another chain, add that network key.

---

## 3. Complexity labels

Create these in GitHub → Issues → Labels (or with the GitHub CLI):

```bash
gh label create "complexity: trivial" --color BFE9F0 --description "Drips Wave — small, well-bounded task"
gh label create "complexity: medium"  --color D7F94B --description "Drips Wave — moderate scope, some design"
gh label create "complexity: high"    --color FF8A3C --description "Drips Wave — large or cross-cutting task"
gh label create "drips-wave"          --color 6C4DF6 --description "Tracked in a Drips Wave program"
gh label create "good first issue"    --color 0E8A16 --description "Good entry point for new contributors"
```

Rules that matter once approved:

- Apply **exactly one** `complexity:*` label per issue, plus `drips-wave`.
- Issues added via GitHub labels default to **Trivial (100 pts)**; raise to
  Medium (150) or High (200) in the Drips app: **Maintainers → Issues**.
- The Wave runs 1 week/month. Review applications **daily** and assign fast.
- Mark issues **Resolved before the Wave ends** so contributors earn points.

---

## 4. Bounty queue (ready to paste)

Each issue is scoped to the existing codebase with explicit acceptance criteria.

### Issue 1 — `good first issue`, `complexity: trivial`
**Title: Fix README accuracy — env vars table, Web UI claim, GHCR reference**
- README's env table is missing `TELEGRAM_CHAT_ID`, `FILTER_ASSETS`,
  `MIN_AMOUNT`, `PORT`, and `MAX_BACKOFF_SECS`, and the top claims a
  "configuration Web UI" that does not exist.
- Also add a short "Verify a webhook signature" snippet showing how a receiver
  checks the `X-Sentinel-Signature` header.
- Acceptance: env table matches `internal/config/config.go`; the Web UI claim is
  removed (or clearly "planned"); signature-verification snippet added; `docs` build clean.

### Issue 2 — `complexity: medium`
**Title: Harden the SSE parser and add unit tests for `internal/horizon`**
- `internal/horizon/stream.go` currently has **no tests**. The parser only
  matches `data: ` (with a space) and doesn't handle multi-line `data:` fields
  or non-payment events (e.g. `create_account`, `path_payment`) that Horizon
  emits on `/payments`.
- Add table-driven tests using `httptest.Server`: single-line events, no-space
  `data:` lines, multi-line data, `"hello"`/`"bye"`, malformed JSON, and
  connection reset → reconnect with backoff. Filter to payment records only.
- Acceptance: accepted events parsed correctly; malformed/foreign events
  skipped; reconnect tested; `go test ./...` and `CGO_ENABLED=0 go vet ./...` pass.

### Issue 3 — `complexity: medium`
**Title: Deduplicate events across SSE reconnects**
- On reconnect, Horizon can redeliver the last event for the stored cursor, so
  the same payment can be dispatched twice (webhook/discord/telegram spam).
- Track a small ring of recently dispatched `event_id`s (e.g. last 1000) and
  skip a payment if its ID was already handled.
- Acceptance: unit test feeds a stream where the cursor repeats the last event
  across two `streamOnce` calls and exactly one dispatch occurs; existing tests
  pass.

### Issue 4 — `complexity: high`
**Title: Persistent cursor checkpointing**
- The cursor lives only in memory (`*lastCursor`), so if the process restarts
  (container restart, deploy) it reconnects with `cursor=now` and **misses every
  payment that landed while it was down** — critical for exchanges/processors.
- Add a checkpoint store (e.g. file under `SENTINEL_CURSOR_FILE`, JSON keyed by
  account) that is atomically written after each dispatched event and loaded on
  startup to resume from the last cursor instead of `now`.
- Acceptance: writing and reloading survives process restart (tested with a
  temp file); write is atomic (temp+rename); corrupt/missing file falls back to `now`;
  `go test ./...` passes.

### Issue 5 — `complexity: medium`
**Title: Retry failed webhook / Discord / Telegram deliveries**
- `dispatch` returns an error and drops the event after a single failed POST
  (log line only). Failed deliveries are silently lost.
- Add bounded retry with exponential backoff (e.g. 3 attempts) per channel and a
  configurable `DISPATCH_RETRIES`, plus a "dead letter" log entry when the
  retries are exhausted.
- Acceptance: unit test asserts a failing server is retried and then reported;
  success path unchanged; config validated for the new option; tests pass.

### Issue 6 — `complexity: medium`
**Title: Publish the container image to GHCR on tag/release**
- README instructs users to run `ghcr.io/astrforgelabs/stellar-sentinel:latest`,
  but nothing builds or publishes that image.
- Add a `.github/workflows/release.yml` that builds the Linux amd64/arm64 images
  from the Dockerfile and publishes to the GHCR tag on every `v*` tag (and a
  `:latest` on main).
- Acceptance: workflow is valid YAML, uses `docker/build-push-action`,
  `permissions: packages: write`, and a snippet in README showing `docker compose pull`.

---

## 5. Suggested application answers

- **"We plan to add issues related to..."** — reliability & ops: persistent
  cursors, retries/dedup for notification delivery, SSE parser hardening +
  tests, GHCR image publishing, and documentation accuracy.
- **Resource links** — repo, README, Actions/CI, Docker/GHCR, Drips Wave page,
  Horizon SSE docs, Stellar docs, org page.

## 6. Requirements

- Go 1.22+ — see `.github/workflows/ci.yml`.
- `go vet ./... && go test ./... && go build -o /dev/null ./cmd/sentinel`