# Deployment Guidance Alignment Feature Plan

## Goal

Tie the development compose stack, production compose stack, and README examples together so contributors can keep them aligned.

## What is required

- A deployment guide that explains the role of each compose file
- Clear contributor guidance on which files to update when runtime shape changes
- README references that make the relationship discoverable

## Plan

1. Add a deployment guide under `docs/`.
2. Explain the responsibilities of the dev compose file, production compose file, and README example.
3. Update README and roadmap references to point at the guide.
4. Keep the contributor guidance aligned with `AGENTS.md`.

## Acceptance criteria

- A deployment guide exists in `docs/`.
- The README points readers to that guide.
- The guide explains how dev, prod, and example compose configurations relate.
- The feature includes examples or concrete references for each deployment artifact.
