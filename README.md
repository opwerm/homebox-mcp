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

## Read only, deliberately

HomeBox accepts `POST` on `/entities` and `/tags`, and `PUT`/`DELETE` on
`/tags/{id}`. None of that is exposed. A model that can rewrite an inventory
whose consequences it cannot see is a different product, and a riskier one.

| tool | what it answers |
|---|---|
| `homebox_list_entities` | search or page through **items** (not locations), filterable by tag or parent |
| `homebox_get_entity` | one entity in full, by id |
| `homebox_entity_tree` | how entities nest — **this is where locations are** |
| `homebox_entity_path` | an entity's ancestors, root first |
| `homebox_list_tags` / `homebox_get_tag` | tags — what older HomeBox called labels |
| `homebox_entity_types` | the types that distinguish an item from a location |
| `homebox_get_asset` | look up by printed asset id rather than UUID |
| `homebox_statistics` | totals, or broken down by location, tag or price |
| `homebox_custom_fields` / `homebox_custom_field_values` | custom field names, and the values in use |
| `homebox_maintenance` | maintenance records, all or for one entity |
| `homebox_bill_of_materials` | the bill-of-materials report |

`/v1/entities` returns items only — locations are reachable through the tree.
The tool descriptions say so, because a model picking a tool reads them: one
that claims to list locations and returns none is worse than no tool at all.

This is not the whole API. HomeBox exposes 42 GET endpoints; these cover the
inventory-reading ones. Deliberately absent: authentication and API-key
management, group administration, bulk export, notifiers, and label/QR
image generation.

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
