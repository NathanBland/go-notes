# Deployment Guide Discoverability Feature Plan

## Goal

Make the deployment guide easy to find from the repository root and README so readers do not have to guess whether deployment documentation exists.

## What is required

- Add a deployment-doc discoverability item to the roadmap
- Create a top-level deployment entry point that readers can find quickly
- Update README links and wording so the deployment docs are obvious
- Keep the deeper deployment guide in sync with the new entry point
- Include examples and documentation updates as part of completion
- Version impact assessment: `patch`
- `CHANGELOG.md` update required: `yes`

## Plan

1. Add the discoverability improvement to the roadmap.
2. Create a root-level deployment guide entry point.
3. Update README links and deployment wording to point at the new top-level guide.
4. Keep the detailed guide in `docs/` aligned with the new entry point.
5. Update the changelog for the new documentation improvement.

## Acceptance criteria

- Readers can find deployment documentation from the repository root without hunting through `docs/`.
- README links point to a deployment guide that is clearly present.
- The deployment entry point includes a clear example and links onward to the detailed guide.
- The roadmap and changelog reflect the documentation improvement.
