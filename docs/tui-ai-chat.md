---
title: "Console AI Chat"
sidebar:
  order: 6
---

Press `[a]` from anywhere in the `datumctl console` to open the AI chat pane. Ask natural-language questions about your resources without leaving the console.

## Prerequisites

An API key from one of the supported LLM providers must be configured:

```
datumctl ai config set anthropic_api_key sk-ant-...
```

See the [AI Assistant](./ai.md) guide for all supported providers and configuration options.

## Layout

The chat pane replaces the main area with a message history sidebar on the left and a conversation panel on the right. The header and status bar remain visible.

```
┌──────────────────────────────────────────────────────────────┐
│ Header (org / project / user / spinner)                      │
├──────────────────┬───────────────────────────────────────────┤
│ Chat History     │ AI Chat                                   │
│ ──────────────── │ ─────────────────────────────────────────  │
│                  │                                           │
│ ▸ list dns zon…  │  You                                      │
│   show project…  │  list dns zones                           │
│                  │                                           │
│                  │  Assistant                                │
│                  │  Here are your DNS zones in web-infra:    │
│                  │    • example.com   (A)    active          │
│                  │    • api.io        (CNAME) active         │
│                  │                                           │
│                  │ ───────────────────────────────────────── │
│                  │ ▸ _                                       │
├──────────────────┴───────────────────────────────────────────┤
│ NORMAL │ [Enter] send  [PgUp/Dn] scroll  [Tab] history nav … │
└──────────────────────────────────────────────────────────────┘
```

## Key bindings

| Key | Action |
|-----|--------|
| `a` | Open / close chat pane |
| `Enter` | Send message |
| `Tab` | Toggle focus between input and history sidebar |
| `j` / `k` | Navigate history sidebar (when sidebar is focused) |
| `PgUp` / `PgDn` | Scroll conversation viewport |
| `n` | New conversation |
| `e` | Export conversation to markdown |
| `y` / `n` | Approve or cancel a pending write operation |
| `Esc` | Return to previous pane |

## Context

The chat agent automatically uses the active console context — the same org and project that the resource table is showing. Switch context with `[c]` as usual and the next chat message will use the updated scope.

## Write operation confirmations

When the assistant proposes a change (creating, updating, or deleting a resource), it pauses and shows an inline confirmation prompt before applying anything:

```
  Assistant
  I'd like to delete DNS zone 'example.com'.

  ⚠  Confirm delete dns_zone 'example.com'?
     Type [y] to approve or [n] to cancel.

  ▸ _
```

Type `y` to proceed or `n` to cancel. The assistant is informed of the decision either way.

## Conversations

Conversations are saved locally and restored when you reopen the console. Use `[n]` to start a fresh conversation, or delete one from the history sidebar. Use `[e]` to export the current conversation to a markdown file.
