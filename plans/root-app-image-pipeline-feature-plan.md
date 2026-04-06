# Root App Image Pipeline Feature Plan

## Goal

Build and publish a production-ready container image for the main `go-notes` API through GitHub Actions.

## What is required

- A production `Dockerfile` for the API
- A GitHub Actions workflow that builds and publishes the image to GHCR
- Clear image naming and tagging rules
- Documentation showing how the image is consumed in production compose files

## Plan

1. Add a multi-stage production Dockerfile for the API.
2. Add a GitHub Actions workflow that builds on pushes and publishes on `main` and version tags.
3. Standardize the image repository and tag output for reuse in production compose.
4. Document how to pull and use the image.

## Acceptance criteria

- A production API Dockerfile exists and builds successfully.
- A GitHub Actions workflow exists for app image builds and publishing.
- The README and deployment docs explain the produced image and expected tags.
- The feature includes examples showing how the image is consumed.
