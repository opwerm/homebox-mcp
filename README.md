# homebox-mcp

Read-only [Model Context Protocol](https://modelcontextprotocol.io) server for
a [HomeBox](https://homebox.software) inventory.

```
oci://ghcr.io/opwerm/homebox-mcp
```

## Why this exists rather than a fork

The obvious starting point, `jeeves5454/Homebox-mcp`, was rejected twice over.

Its **licence is asserted but never granted** — the README and `package.json`
both say MIT, but there is no LICENSE file, GitHub's licence API returns 404,
and no copyright holder is named. The author's intent is clear; intent is not
a grant, and redistribution is the part that needs one.

And it **targets an API that no longer exists**. HomeBox 0.26.2 replaced
`/v1/items`, `/v1/labels` and `/v1/locations` with `/v1/entities` and
`/v1/tags`. Every tool it exposes would 404.

## What it exposes

The inventory is covered end to end: entities, tags, entity types, templates,
maintenance records and attachments all read, create, update and delete.

| tools | what they cover |
|---|---|
| `homebox_list_entities` `homebox_get_entity` `homebox_create_entity` `homebox_update_entity` `homebox_patch_entity` `homebox_delete_entity` `homebox_duplicate_entity` | entities — **items, not locations** |
| `homebox_entity_tree` `homebox_entity_path` | how entities nest — **this is where locations are** |
| `homebox_list_tags` `homebox_get_tag` `homebox_create_tag` `homebox_update_tag` `homebox_delete_tag` | tags — what older HomeBox called labels |
| `homebox_entity_types` `homebox_create_entity_type` `homebox_update_entity_type` `homebox_delete_entity_type` | the types that distinguish an item from a location |
| `homebox_list_templates` `homebox_get_template` `homebox_create_template` `homebox_update_template` `homebox_delete_template` `homebox_create_item_from_template` | templates |
| `homebox_maintenance` `homebox_create_maintenance` `homebox_update_maintenance` `homebox_delete_maintenance` | maintenance records |
| `homebox_get_attachment` `homebox_update_attachment` `homebox_delete_attachment` | attachments on an entity |
| `homebox_get_asset` `homebox_search_barcode` | look up by printed asset id or barcode |
| `homebox_statistics` `homebox_bill_of_materials` `homebox_currencies` | totals, the CSV report, and the currency list |
| `homebox_custom_fields` `homebox_custom_field_values` | custom field names, and the values in use |
| `homebox_get_configuration` `homebox_set_configuration` | group and user settings, as one document each way |

Every tool carries MCP annotations, so a client can tell a read from a delete
without calling it: reads are `readOnlyHint`, creates are non-destructive, and
updates and deletes are `destructiveHint`.

`/v1/entities` returns items only — locations are reachable through the tree.
The tool descriptions say so, because a model picking a tool reads them: one
that claims to list locations and returns none is worse than no tool at all.

## What it will not do

Some of the API is deliberately unreachable, because these are the calls whose
damage a model cannot see coming and cannot undo:

- **`/actions/*`** — including `ensure-asset-ids`, `zero-item-time-fields` and
  `wipe-inventory`. One of these empties the group.
- **Authentication and API keys** — minting or revoking a credential is not
  inventory work.
- **Group membership and invitations** — who can see the data is not the
  model's call.
- **Bulk import and export** — an import is an undoable mass write.
- **Notifiers** — `set_configuration` will not touch them, so a misread
  instruction cannot redirect or silence alerts.

Configuration is two tools rather than a dozen: one read that returns the group
and user settings together, and one write that takes the same shape back.

## Authentication

HomeBox accepts exactly one credential: **its own API key**, as
`Authorization: Bearer`. It does not accept an OIDC token — its OIDC support
is a browser redirect flow that ends by *issuing* one of these keys.

Mint one under user settings, then:

```
HOMEBOX_URL=http://homebox HOMEBOX_TOKEN=hb_... homebox-mcp
```

Data in HomeBox is scoped by **group**, not by user, so the key sees whatever
its owner's group contains. Everyone in one group sees one inventory, which
for a household is the point rather than a limitation.

## Transports

```
TRANSPORT=stdio   default; run it next to a local client
TRANSPORT=http    streamable HTTP on ADDR (default 0.0.0.0:8080)
```

`0.0.0.0`, not loopback: in a container, binding to `127.0.0.1` means nothing
can reach it, including the readiness probe.

**The HTTP transport authenticates nothing.** It is built to sit behind a
gateway that validates a token. Exposed directly to a network, it is the
whole inventory, unauthenticated. `/healthz` is liveness only and does not
call HomeBox — a probe that fails when a dependency blips restarts a process
that would have recovered.

## Licence

MIT, © 2026 Oleg Tsarev.
