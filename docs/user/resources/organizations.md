# Organizations

This section details the `datumctl` commands used to interact with Datum Cloud
Organizations.

## List organizations

To list the Datum Cloud organizations your authenticated user has access to:

```bash
datumctl organizations list [--output <format>]
```

*   `--output <format>`: (Optional) Specify the output format. Supported
    options are `table` (default), `json`, `yaml`.

This command fetches the list of organizations from the Datum Cloud API using
your active authenticated session.
