# MCP Build Pipeline Feature Plan

## Goal

Build and publish the MCP server as a distributable artifact and container-friendly runtime through GitHub Actions.

## What is required

- A production-oriented MCP Dockerfile
- A GitHub Actions workflow for MCP packaging and release automation
- Clear guidance on when to use the packaged binary versus the container image
- Documentation for downstream MCP users

## Plan

1. Add a production Dockerfile for `cmd/mcp`.
2. Add a GitHub Actions workflow that validates snapshot builds on pull requests and publishes release artifacts on tags.
3. Build or publish an MCP image path that can be consumed outside local development.
4. Update docs with install and usage guidance.

## Acceptance criteria

- An MCP runtime Dockerfile exists and builds successfully.
- A GitHub Actions workflow exists for MCP build and release automation.
- The docs explain how the MCP binary and/or image should be consumed.
- The feature includes examples showing how the MCP build outputs are used.
