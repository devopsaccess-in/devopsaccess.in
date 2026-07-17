## Context

<!-- Why this change? Link the Notion task / issue if there is one. -->

## Changes

<!-- What changed, at the level a reviewer needs. -->

## Testing

<!-- Unit/integration/e2e evidence. Paste terraform plan output for infra PRs. -->

## Risks & rollback

<!-- Blast radius and how to undo (revert commit / redeploy previous build). -->

## Checklist

- [ ] `FEATURES.md` updated (any user-facing behaviour added/changed/removed)
- [ ] E2E coverage in `e2e/` added or updated for the feature rows this touches
- [ ] `go test ./...` green in touched modules; `pnpm typecheck` green for touched apps
- [ ] No new paid services / stays within the ₹10k budget (`cost-check` if unsure)
