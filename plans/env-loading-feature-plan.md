# Env Loading Feature Plan

## Goal

Make local developer commands load `.env` and `.env.local` automatically so running the API and MCP server matches developer expectations.

## What is required

- Update the Makefile so local run targets source env files automatically
- Keep precedence clear, with `.env.local` overriding `.env`
- Update the README so the behavior is documented

## Implementation plan

1. Add a small reusable shell snippet in the Makefile to source `.env` and `.env.local` when present.
2. Apply that loader to `make run` and `make run-mcp`.
3. Update the README to explain that local run targets auto-load env files.

## Acceptance criteria

- `make run` reads values from `.env.local` without requiring manual `source`
- `make run-mcp` reads values from `.env.local` without requiring manual `source`
- `.env.local` overrides `.env` when both files exist
- The README documents the behavior
