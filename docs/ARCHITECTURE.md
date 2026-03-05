# Architecture

> This doc is the target output of the `architect` Claude agent.

## Goals
- Go GUI (cross-platform)
- Rust physics + rendering acceleration via stable FFI
- Optional ray tracing
- Headless CLI
- Installer builds for macOS/Linux/Windows

## High-Level Diagram

```mermaid
flowchart LR
  UI[Go UI Panels] -->|commands| CTRL[Controller]
  CTRL -->|step dt| SIM[Sim API]
  SIM -->|FFI| PHYS[Rust physics_core]
  CTRL --> RENDER[Render API]
  RENDER -->|FFI| GPU[Rust render_core (wgpu)]
  GPU --> UI
  CTRL --> VALID[Validation/Scenarios]
  CTRL --> LAUNCH[Launch Planner]
```

## Module Responsibilities
_TODO_
