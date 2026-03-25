# Holocron — Design Principles

> These are non-negotiable. Every design decision must pass through these filters.

## 1. Source Independence

Each provider (Claude Code, OpenClaw, Codex) is an **independent stream**. No provider knows about or depends on another. Adding a new source means writing one adapter — nothing else changes.

**Test:** Can you delete a provider package and everything else still compiles? If yes, this principle holds.

## 2. Separation of Concerns

Four boundaries that must never blur:

| Layer | Responsibility | Knows about |
|-------|---------------|-------------|
| **Provider** | Collect events from one source | Its own source only |
| **EventBus** | Route events between layers | Nothing — just channels |
| **Store** | Persist and query events | Event schema only |
| **Renderer** | Display to human | `<-chan MonitorEvent` only |

A Provider never imports TUI code. The TUI never imports Provider code. They communicate only through the EventBus.

**Test:** Draw the import graph. Do arrows only point inward (toward `collector/types`)? If any arrow points outward, something is wrong.

## 3. View-Layer Linking

"R2 is using this Claude Code session" is **NOT** a data layer fact. It is a **view layer annotation** (labels). The data layer only knows: "a Claude Code session exists" and "an OpenClaw session exists" — independently.

**Test:** Remove all labels. Does the data still make sense on its own? If yes, this principle holds.

## 4. Config Over Magic

Holocron does not guess which sources exist. You declare them in config. Within each declared source, auto-discovery finds sessions.

Explicit at the macro level. Automatic at the micro level.

**Test:** If a source is not in config, Holocron must not attempt to connect to it — even if it's running on the machine.

## 5. Channel-Based Decoupling

The TUI receives `<-chan MonitorEvent`. Whether that channel comes from an in-process Collector or a remote WebSocket — the TUI cannot tell and does not care.

This is what enables the on-demand → daemon transition without code changes.

**Test:** Can you swap the in-process EventBus subscription for a WebSocket client, with zero changes to TUI code? If yes, this principle holds.

## 6. No Premature Abstraction

Build what you need for the current phase. Don't build generic plugin systems, RPC frameworks, or extensibility hooks until a second use case proves they're needed.

The interfaces (Provider, Store, EventBus) are the abstraction — that's enough.

**Test:** Is there code that exists "just in case" or "for future flexibility" with no current user? Delete it.
