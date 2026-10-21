# Project template history

`datumctl apps templates` reads immutable project template revisions, compares them with current authoring, restores a revision, and deletes authoring with exact identity preconditions. These commands require the Apps authoring API in the selected project. They do not install Apps or publish templates.

Sign in and inspect the current authoring template:

```sh
datumctl login
datumctl get apptemplates.apps.miloapis.com web-api --project my-project -o json
```

The response includes the template's `metadata.uid` and `metadata.resourceVersion`. A recreated name has a different UID. Save these values when reviewing a deletion.

List history and compare a revision:

```sh
datumctl apps templates history web-api --project my-project
datumctl apps templates diff web-api --revision TEMPLATE_UID.REVISION_UID --project my-project
```

History prints JSON containing revision names, sequence numbers, digests, actor UIDs and acceptance times. It excludes definition values. The diff prints the current definition and selected revision as a unified diff, so handle it with the same care as the template source. No server writes occur.

Validate a restore, then restore the selected revision:

```sh
datumctl apps templates restore web-api --revision TEMPLATE_UID.REVISION_UID --dry-run=server --project my-project
datumctl apps templates restore web-api --revision TEMPLATE_UID.REVISION_UID --project my-project
```

A restore reads the current template and the exact selected revision, checks they share the same template UID, and writes the old definition as a new revision with the current UID and resource version. It preserves all earlier history. It cannot restore a revision from an old incarnation into a replacement template with the same name. A concurrent update returns a conflict without retrying. Server dry-run is advisory and does not reserve the template for the later restore.

Validate deletion with the UID and version you inspected, then repeat without the dry-run flag to delete authoring:

```sh
datumctl apps templates delete web-api --uid TEMPLATE_UID --resource-version VERSION --dry-run=server --project my-project
datumctl apps templates delete web-api --uid TEMPLATE_UID --resource-version VERSION --project my-project
```

The delete command puts both preconditions and dry-run in its JSON request body. The authoring API rejects generic delete requests without both preconditions, and rejects DELETE query options. It closes the exact UID, frees the name, and retains immutable history. The command reports JSON with the returned name, UID, resource version, generation, action and dry-run state.

Read retained history after deletion by UID:

```sh
datumctl apps templates history --uid TEMPLATE_UID --project my-project
```

An unknown UID has empty history. The command requires project revision-list access; it does not bypass authorization or expose another project's revisions.

A missing API or resource produces an actionable error. A permission denial asks you to check the signed-in account and project access. A conflict requires inspection and a fresh decision; the command never substitutes a replacement UID or retries a newer version. If a write response is lost, malformed, or fails after the server may have committed, the command reports an unknown outcome. Inspect the template and history before retrying a restore, because a second successful restore creates another revision.

The current Apps authoring service supports bounded lists without pagination and does not accept installation graphs. These commands do not change those limits. Full login-to-Apps behavior under production IAM and the private/published installation flows remain separate integration work.
