# Go For Node Developers

## Optional fields

In JavaScript it is common to use loose objects and check whether a property exists at runtime.

In Go, optional fields are usually represented with pointers:

- `*string` for nullable or optional strings
- `*bool` for optional booleans
- `*time.Time` for optional timestamps

That is why `title` and `share_slug` are pointers in the note model.

## PATCH requests

A pointer alone is not enough for PATCH when `null` is meaningful.

Example:

- field omitted
- field present with a string value
- field present with `null`

The project uses pointer fields plus explicit `Set` flags so the code can tell those cases apart.

## UTC timestamps

Go's `time.Time` carries location data.

This project stores timestamps in PostgreSQL `timestamptz`, sets the DB session timezone to UTC, and serializes JSON as RFC3339 timestamps. That keeps the API stable across machines and avoids “works on my laptop” timezone bugs.

## Context

Go request handlers pass `context.Context` to everything downstream. Think of it as request-scoped cancellation and metadata, not as a general-purpose global store.
