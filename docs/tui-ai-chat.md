# TUI AI Chat Pane — Enhancement Design

## Overview

This document describes the design for surfacing the AI agent inside the `datumctl console` TUI as an interactive chat pane. Users can ask natural-language questions about their Datum Cloud resources without leaving the console.

The implementation builds on `internal/ai/` (the existing AI assistant package) by wiring `Agent.RunTurn()` and `TUIGate` into the Bubbletea model.

---

## UX Design

### Layout

Press `[a]` from anywhere in the console to open the chat pane. The standard header and status bar remain visible; the main area splits into a message-history sidebar (left) and a conversation panel (right):

```
┌──────────────────────────────────────────────────────────────┐
│ Header (org / project / user / spinner)                      │
├──────────────────┬───────────────────────────────────────────┤
│ Chat History     │ AI Chat                                   │
│ ──────────────── │ ─────────────────────────────────────────  │
│                  │                                           │
│ ▸ list dns zon…  │  You                                      │
│   show project…  │  list dns zones                           │
│   delete exampl… │                                           │
│                  │  Assistant                                │
│                  │  Here are your DNS zones in web-infra:    │
│                  │    • example.com   (A)    active          │
│                  │    • api.io        (CNAME) active         │
│                  │                                           │
│                  │  ⚠  Confirm delete 'example.com'? [y/n]  │
│                  │ ───────────────────────────────────────── │
│                  │ ▸ _                                       │
├──────────────────┴───────────────────────────────────────────┤
│ NORMAL │ [Enter] send  [PgUp/Dn] scroll  [Tab] history nav … │
└──────────────────────────────────────────────────────────────┘
```

### Key Bindings

| Key | Context | Action |
|-----|---------|--------|
| `a` | Any pane | Open / close chat pane (toggles back to origin) |
| `Enter` | Chat input focused | Send message |
| `Tab` | Chat pane | Toggle focus: input ↔ sidebar |
| `j` / `k` | Sidebar focused | Move cursor; viewport scrolls to that message |
| `PgUp` / `PgDn` | Chat pane | Scroll conversation viewport |
| `y` / `n` | Confirm pending | Approve or deny a proposed write operation |
| `c` | Chat pane | Open context switcher |
| `?` | Chat pane | Open help overlay |
| `a` / `Esc` | Chat pane | Return to origin pane |
| `q` / `Ctrl+C` | Chat pane | Quit |

### Focus Model

The chat pane has two internal focus sub-modes, toggled with `Tab`:

**Input focused (default)**
- Status bar: `NORMAL │ [Enter] send  [PgUp/Dn] scroll  [Tab] history nav  [a] back  …`
- Keyboard input goes to the text field; `j`/`k` type characters normally.

**Sidebar focused**
- Status bar: `HISTORY │ [j/k] move  [Tab] back to input  [a] exit  [q] quit`
- `j`/`k` navigate the message history list; the right viewport auto-scrolls to the selected message.

### Write Operation Confirmation

When the AI proposes a mutating operation (apply, delete), the `TUIGate` pauses the agent goroutine and appends an inline confirmation prompt to the conversation. The input area accepts `y` or `n`:

```
  Assistant
  I'd like to delete DNS zone 'example.com'.

  ⚠  Confirm delete dns_zone 'example.com'?
     Type [y] to approve or [n] to cancel.

  ▸ _
```

- `y` → approves; agent continues
- `n` / `Esc` → declines; agent skips the operation and explains
- The status bar switches to overlay mode while confirmation is pending: `[y] approve  [n] cancel`

---

## Scope

### Context resolution

The chat agent uses the active console context (same org/project that the resource table uses). No separate AI config for the context is needed; it inherits from whatever `datumctl ctx use` set.

The API key still comes from `~/.config/datumctl/ai.yaml` (set via `datumctl ai config set anthropic_api_key …`). If no key is configured, the pane opens with an error message and a hint to run the config command.

### Message history sidebar

The left column repurposes the sidebar slot to show the current session's user messages as a compact, scrollable list. Selecting a message scrolls the conversation viewport to that point. History is **session-only** — it is not persisted across console launches (persistence is out of scope for v1).

### Conversation state

Each `[a]` open/close cycle **preserves** the conversation (the agent's history slice is kept on the AppModel). Pressing `[a]` again re-enters the same conversation. To clear history, the user can restart the console.

---

## Implementation Plan

### New files

| File | Purpose |
|------|---------|
| `internal/console/components/chatpane.go` | Right-side conversation panel (viewport + textinput + spinner) |
| `internal/console/components/chatsidebar.go` | Left-side message history list |

### Modified files

| File | Changes |
|------|---------|
| `internal/console/model.go` | `ChatPane` PaneID; `chat`, `chatSidebar`, `chatAgent`, TUIGate channel fields; `handleChatKey()`; `[a]` key in `handleNormalKey()`; View/updatePaneFocus/recalcLayout/NewAppModel updates |
| `internal/console/components/statusbar.go` | `ModeSidebarNav` constant; `"CHAT"` pane hints |
| `internal/ai/agent.go` | `SetGate(ConfirmGate)` one-line setter to refresh gate context per turn |

### New message types (in model.go)

```go
type chatAgentInitMsg  struct { agent *datumai.Agent; err error }
type chatResponseMsg   struct { response string; err error }
type chatConfirmReqMsg struct { req datumai.ConfirmRequest }
type chatConfirmDoneMsg struct{}
```

### New tea.Cmd functions (in model.go)

```go
// Loads AI config, builds Agent with TUIGate; called on first [a] press.
func initChatAgentCmd(ctx context.Context, factory *client.DatumCloudFactory,
    confirmCh chan<- datumai.ConfirmRequest, orgID, projectID, namespace string,
    turnCtx context.Context) tea.Cmd

// Wraps agent.RunTurn in a goroutine.
func sendChatMessageCmd(ctx context.Context, agent *datumai.Agent, text string) tea.Cmd

// Blocks until TUIGate sends a confirmation request or ctx is cancelled.
func listenForConfirmCmd(ctx context.Context, ch <-chan datumai.ConfirmRequest) tea.Cmd
```

### ChatPaneModel fields

```go
type ChatPaneModel struct {
    width, height      int
    focused            bool
    processing         bool    // true while RunTurn goroutine is running
    agentReady         bool
    agentErr           string
    confirmPending     bool    // true when waiting for y/n

    messages           []chatMsg      // full conversation history
    messageLineOffsets []int          // viewport line where each user message starts
    input              textinput.Model
    vp                 viewport.Model
    sp                 spinner.Model
}
```

### AppModel new fields

```go
chat             components.ChatPaneModel
chatSidebar      components.ChatSidebarModel
chatOriginPane   DashboardOrigin
chatAgent        *datumai.Agent
chatConfirmCh    chan datumai.ConfirmRequest // buffered(1); shared with TUIGate
chatConfirmReply chan bool                   // non-nil while confirm is pending
chatTurnCtx     context.Context
chatTurnCancel  context.CancelFunc
```

---

## Out of Scope (v1)

- Chat session persistence across console launches
- Streaming tokens to viewport as they arrive (`RunTurn` buffers the full response)
- Help overlay `[a] AI chat` shortcut hint
- Multiple concurrent chat sessions

---

## Prerequisites

An Anthropic (or OpenAI / Gemini) API key must be set before the pane can initialize:

```sh
datumctl ai config set anthropic_api_key sk-ant-...
```

The pane opens regardless and shows a clear error message with this hint if no key is found.
