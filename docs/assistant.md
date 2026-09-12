---
title: "AI Assistant"
sidebar:
  order: 5
---

Patch is the Datum Cloud assistant. Ask it about your resources in plain
language, and it answers using the same permissions your account already has —
no API key of your own, and nothing to configure.

Patch ships as a plugin rather than as part of `datumctl`, so it updates on its
own schedule:

```
datumctl plugin install assistant
```

## Starting a conversation

```
# Full-screen chat
datumctl assistant

# Ask one question and exit
datumctl assistant chat "Why is the api-backend workload not available?"

# Pick up where you left off
datumctl assistant resume
```

Conversations are held by the assistant service, not on your machine, so you can
continue one from anywhere you are logged in. Browse them with
`datumctl assistant conversations`.

## Making changes

Patch proposes changes as a plan: the exact manifests it intends to apply,
shown before anything happens. Applying a plan is a separate, explicit step, and
the service refuses any manifest that differs from what you were shown.

## Moving from `datumctl ai`

`datumctl ai` was a local agent that needed your own Anthropic, OpenAI, or
Gemini key. It has been removed. The command still works and now runs the
plugin, so existing scripts keep going; the provider key and its configuration
are no longer used and can be deleted.

The `[a]` chat pane in `datumctl console` ran on the same local agent and has
also been removed. Use `datumctl assistant` for chat.
