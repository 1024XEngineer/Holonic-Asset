<h1 align="center">Holonic Asset</h1>

<p align="center">
  <img src="docs/image/logo-dark.svg" alt="Holonic Asset logo" width="240">
</p>

<p align="center"><strong>AI-assisted asset production for coherent game worlds.</strong></p>

<p align="center">
  <a href="https://codecov.io/gh/1024XEngineer/Holonic-Asset"><img src="https://codecov.io/gh/1024XEngineer/Holonic-Asset/branch/main/graph/badge.svg" alt="Project test coverage"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License: Apache 2.0"></a>
</p>

<p align="center">
  <a href="https://www.holonicasset.xyz/">Website</a> ·
  <a href="#what-you-can-do-today">Capabilities</a> ·
  <a href="#project-status">Project status</a> ·
  <a href="#getting-started">Getting started</a>
</p>

Holonic Asset brings AI generation into a project-based game asset workflow. It helps creators generate, organize, iterate on, version, and export related assets without reducing a game world to a collection of disconnected images.

The project is built for indie game developers, small game teams, prototype developers, and pixel-art creators who need a repeatable path from a visual direction to usable game assets.

## What You Can Do Today

### Build a shared project context

- Create a project from a guided brief, an initial idea, or a blank starting point.
- Define the game type, visual style, target platform, camera perspective, and reference images.
- Generate and regenerate a project preview that captures the intended gameplay and art direction.
- Carry the same project context into subsequent asset generation.

### Generate game assets

- Generate and revise multi-direction character prototypes.
- Create character animations and regenerate selected animation frames.
- Generate and revise game objects while preserving their visual identity.
- Generate scenery as separate, composable layers.
- Generate tilesets, add new items, and revise an item or selected tiles.
- Inspect queued generation work and retry or remove failed runs.

### Organize an asset library

- Manage generated characters, objects, scenery, and tilesets inside a project.
- Filter the library by asset type, name, version, or tags.
- Edit asset names, descriptions, and tags.
- Open an existing asset and continue working from its current state.

### Iterate without losing previous work

- Review generated candidates before applying them to an asset.
- Continue editing an existing result instead of starting over.
- Use a canvas-based workspace for sprites, scenery, and tilesets.
- Undo and redo changes made during an editing session.
- Save confirmed changes as a new asset version.

### Review saved versions

- Browse the revision history of an asset.
- See which saved version is currently active.
- Keep a traceable record of confirmed generation and editing decisions.

### Export asset packages

- Export Character, Object, Scenery, and Tileset assets.
- Download a ZIP package containing PNG resources, structured asset data, and a manifest.
- Export the asset's current saved version rather than an unconfirmed draft.

## Supported Workflows

| Asset type | Current support |
| --- | --- |
| Character | Multi-direction prototype generation, iterative editing, animation generation, frame editing, version history, and package export |
| Object | Prototype generation, iterative editing, version history, and package export |
| Scenery | Layered generation, canvas composition, version history, and package export |
| Tileset | Generation, item creation, item and tile editing, version history, and package export |

## Project Status

Holonic Asset is under active development. The core game-asset production workflow is already implemented, but the project is not yet a complete SaaS product.

The following SaaS capabilities are not currently available:

- Self-service sign-up, email verification, and password recovery
- Organizations, teams, member invitations, and collaborative permissions
- Subscription plans, payments, invoices, usage quotas, and credit management
- Shared review, comments, approval flows, and user notifications
- A complete administration experience for operating a hosted service

The current authentication flow expects accounts to be provisioned by the operator. The project should be treated as an actively developed application rather than a production-ready multi-tenant SaaS offering.

## Core Concepts

### Project

A Project represents a game, prototype, or consistently themed asset pack. It holds the shared creative context used to organize and guide asset production.

### Asset

An Asset is an independently editable and deliverable unit within a Project, such as a character, object, scene, tileset, interface, or audio track. Each asset type retains the structure needed by its workflow.

### Record

A Record is a saved snapshot of an Asset after a generation or edit is confirmed. Records make creative history traceable and allow an asset to return to an earlier state.

![Asset Domain Model](./docs/image/Asset%20Domain%20Model.svg)

## Getting Started

Follow the [Quick Start guide](./docs/en/quick-start.md) to run Holonic Asset locally.

For problems and feature requests, use the repository's [GitHub issue templates](https://github.com/1024XEngineer/Holonic-Asset/issues/new/choose).

## License

Holonic Asset is licensed under the [Apache License 2.0](./LICENSE).
