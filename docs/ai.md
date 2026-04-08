---
title: "AI Assistant"
sidebar:
  order: 5
---

`datumctl ai` is a natural-language interface for Datum Cloud. Describe what you
want to do in plain English and the assistant translates it into resource
operations — listing, inspecting, creating, updating, and deleting resources —
with confirmation prompts before any changes are applied.

## Prerequisites

You need an API key from one of the supported LLM providers:

| Provider  | Environment variable  | Get a key at                        |
|-----------|-----------------------|-------------------------------------|
| Anthropic | `ANTHROPIC_API_KEY`   | console.anthropic.com               |
| OpenAI    | `OPENAI_API_KEY`      | platform.openai.com                 |
| Gemini    | `GEMINI_API_KEY`      | aistudio.google.com                 |

You also need to be logged in to Datum Cloud:

```
datumctl auth login
```

## Quick start

The fastest way to get started is to save your API key and default organization
once, then never pass flags again:

```
# Save your API key
datumctl ai config set anthropic_api_key sk-ant-...

# Save your default organization or project
datumctl ai config set organization my-org-id
# or
datumctl ai config set project my-project-id

# Now just run it
datumctl ai "what resources do I have?"
```

Find your organization ID with:

```
datumctl get organizations
```

## Usage

```
datumctl ai [query] [flags]
```

### Single query

```
datumctl ai "list all DNS zones" --project my-project-id
```

### Interactive session (REPL)

Omit the query argument to start a conversation. The assistant remembers
context across turns so you can ask follow-up questions.

```
datumctl ai --organization my-org-id
```

Type `exit` or `quit` to end the session.

### Pipe mode

```
echo "how many projects do I have?" | datumctl ai --organization my-org-id
```

Read-only operations work in pipe mode. Write operations (apply, delete) are
automatically declined — run interactively to apply changes.

## Configuration

`datumctl ai config` manages a configuration file that stores defaults for every
`datumctl ai` invocation. On Linux/macOS the file is at
`~/.config/datumctl/ai.yaml`; on Windows it is at `%AppData%\datumctl\ai.yaml`.

### Set a value

```
datumctl ai config set <key> <value>
```

| Key                 | Description                                              |
|---------------------|----------------------------------------------------------|
| `organization`      | Default organization ID                                  |
| `project`           | Default project ID (mutually exclusive with organization)|
| `namespace`         | Default namespace (default: `default`)                   |
| `provider`          | LLM provider: `anthropic`, `openai`, or `gemini`         |
| `model`             | Model name, e.g. `claude-sonnet-4-6`, `gpt-4o`          |
| `max_iterations`    | Agentic loop iteration cap (default: `20`)               |
| `anthropic_api_key` | Anthropic API key                                        |
| `openai_api_key`    | OpenAI API key                                           |
| `gemini_api_key`    | Gemini API key                                           |

### Show current configuration

API keys are redacted in the output.

```
datumctl ai config show
```

### Remove a value

```
datumctl ai config unset organization
```

### Priority order

Later entries override earlier ones:

1. Config file (`~/.config/datumctl/ai.yaml`)
2. CLI flags (`--organization`, `--model`, etc.)
3. Environment variables (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`)

## Flags

| Flag               | Description                                        |
|--------------------|----------------------------------------------------|
| `--organization`   | Organization context (overrides config file)       |
| `--project`        | Project context (overrides config file)            |
| `--namespace`      | Default namespace (overrides config file)          |
| `--model`          | Model override, e.g. `claude-sonnet-4-6`, `gpt-4o`|
| `--max-iterations` | Agentic loop iteration cap (default: `20`)         |

## How it works

The assistant has access to the following tools, which map directly to the same
operations available via `datumctl get`, `datumctl apply`, and the MCP server:

| Tool                  | Operation                              | Requires confirmation |
|-----------------------|----------------------------------------|-----------------------|
| `list_resource_types` | Discover available resource types      | No                    |
| `get_resource_schema` | Fetch the schema for a resource type   | No                    |
| `list_resources`      | List resources of a given kind         | No                    |
| `get_resource`        | Get a single resource by name          | No                    |
| `validate_manifest`   | Server-side dry-run validation         | No                    |
| `apply_manifest`      | Create or update a resource            | **Yes**               |
| `delete_resource`     | Delete a resource                      | **Yes**               |
| `change_context`      | Switch organization/project/namespace  | No                    |

Read operations execute immediately. For `apply_manifest` and `delete_resource`
the assistant shows a preview of the proposed change and prompts:

```
--- Proposed action ---
Tool:    apply_manifest
Details:
{
  "yaml": "apiVersion: ..."
}
-----------------------
Apply changes? [y/N]:
```

Type `y` to proceed. Any other input cancels the operation — the assistant is
informed it was skipped and will ask what to do next.

## Provider selection

The provider is chosen automatically from whichever API key is available.
When multiple keys are present the priority is Anthropic → OpenAI → Gemini.

Override with `--model` using a provider-prefixed model name:

```
datumctl ai "list zones" --model claude-opus-4-6      # Anthropic
datumctl ai "list zones" --model gpt-4o               # OpenAI
datumctl ai "list zones" --model gemini-2.0-flash      # Gemini
```

Or set a permanent default:

```
datumctl ai config set model claude-sonnet-4-6
```

## Default models

| Provider  | Default model        |
|-----------|----------------------|
| Anthropic | `claude-sonnet-4-6`  |
| OpenAI    | `gpt-4o`             |
| Gemini    | `gemini-2.0-flash`   |
