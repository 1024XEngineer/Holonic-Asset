<h1 align="center">Holonic Asset</h1>

<p align="center">
  <img src="docs/image/logo-dark.svg" alt="Holonic Asset logo" width="240">
</p>

<p align="center"><strong>AI-assisted asset production for coherent game worlds.</strong></p>

<p align="center">
  <a href="https://github.com/1024XEngineer/Holonic-Asset/actions/workflows/frontend-ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/1024XEngineer/Holonic-Asset/frontend-ci.yml?branch=main&label=frontend%20CI" alt="Frontend CI status"></a>
  <a href="https://github.com/1024XEngineer/Holonic-Asset/actions/workflows/backend-ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/1024XEngineer/Holonic-Asset/backend-ci.yml?branch=main&label=backend%20CI" alt="Backend CI status"></a>
  <a href="https://codecov.io/gh/1024XEngineer/Holonic-Asset"><img src="https://codecov.io/gh/1024XEngineer/Holonic-Asset/branch/main/graph/badge.svg?flag=backend" alt="Backend test coverage"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License: Apache 2.0"></a>
</p>

<p align="center">
  <a href="#tech-stack">Tech stack</a> ·
  <a href="#core-capabilities">Capabilities</a> ·
  <a href="#domain-model">Domain model</a> ·
  <a href="#roadmap">Roadmap</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26.5-00ADD8?logo=go&logoColor=white" alt="Go 1.26.5">
  <img src="https://img.shields.io/badge/React-19.2.7-149eca?logo=react&logoColor=white" alt="React 19.2.7">
  <img src="https://img.shields.io/badge/TypeScript-6.0.x-3178C6?logo=typescript&logoColor=white" alt="TypeScript 6.0.x">
  <img src="https://img.shields.io/badge/Vite-8.x-646CFF?logo=vite&logoColor=white" alt="Vite 8.x">
  <img src="https://img.shields.io/badge/pnpm-11.5.0-F69220?logo=pnpm&logoColor=white" alt="pnpm 11.5.0">
</p>

Holonic Asset brings AI generation into real-world game development workflows. Within a shared project context, creators can generate, iterate on, connect, manage, and export characters, objects, scenes, tilesets, and UI Set assets instead of ending up with isolated images.

## Tech Stack

| Area | Technology | Version / source |
| --- | --- | --- |
| Frontend | React + TypeScript | React 19.2.7 · TypeScript 6.0.x |
| Frontend tooling | Vite + Vitest | Vite 8.x · Vitest 4.1.10 |
| Frontend package manager | pnpm | 11.5.0 (`packageManager`) |
| Backend | Go + Echo + GORM | Go 1.26.5 · Echo 4.15.4 · GORM 1.31.2 |
| API contract | OpenAPI + Huma | Generated from `openapi/openapi.json` |
| Quality | GitHub Actions + Codecov | Frontend and backend CI/coverage workflows |

## Who It Is For

Holonic Asset is built for:

- Indie game developers and small game teams
- Prototype developers who need to build game demos quickly
- Pixel art and character design enthusiasts

## Why Holonic Asset

General-purpose image generation tools often fail to address the key challenges of game asset production: assets lack a shared project context, revision history is difficult to track, and generated results still need to be sliced and converted before they can be imported into a game engine.

Holonic Asset organizes asset production into a continuous workflow:

Project setup → Asset generation and management → Continuous iteration → Batch processing → Export and game integration

Project-level visual direction, game genre, perspective, target platform, and reference images form a shared context that helps keep characters, scenes, UI Set, and objects consistent in style, proportions, and color palette.

## Core Capabilities

- **Project context**: Centrally manage visual direction and generation settings for a game, prototype, or themed asset pack.
- **Multiple asset types**: Support characters, objects, UI Set, scenes, and tilesets.
- **Continuous creation**: Repeatedly generate, partially redraw, and manually refine the same asset without losing its creative history.
- **Version history**: Create a Record whenever a generation or edit is confirmed, making it possible to review the history and restore any version.
- **Asset relationships and tags**: Track relationships between characters, animations, sound effects, scenes, and other assets while supporting search and batch operations.
- **Production-ready exports**: Export PNG, GIF, spritesheet, tileset, JSON, and other formats according to asset type, with subsequent conversion for game engines such as Unity and Godot.

## Domain Model

![Asset Domain Model](./docs/image/Asset%20Domain%20Model.svg)

### Project

A Project represents a game, game prototype, or consistently themed asset pack. It stores information such as the game genre, visual style, target platform, description, camera perspective, and reference images while providing shared AI context and centralized asset management.

### Asset

An Asset is an independently created, iterated, and deliverable unit within a Project. A character, wooden chest, inventory interface, scene, or background music track can each be an Asset. Every asset type retains the structure it needs, such as character prototypes and animation frames, scene layers, or placeable Items and Tiles within a Tileset.

### Record

A Record is a complete snapshot of an Asset after a creation or edit is confirmed. It makes generation history traceable, comparable, and reversible so users can confidently explore different creative directions.

## Roadmap

The first milestone focuses on completing the core workflow:

- [ ] Web application for asset creation and management
- [ ] Project creation, editing, and global visual configuration
- [ ] Asset creation, search, duplication, editing, and deletion
- [ ] AI regeneration and asset content editing
- [ ] Animation generation from character or object prototypes
