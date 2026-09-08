# Revision preconditions

Existing subscriptions, notification channels/templates, categories, payment
methods, currencies and backup destinations expose a positive `revision`.
Updates require it in the JSON body; deletes and icon uploads require
`?revision=N`. Creation does not require a revision. Successful writes increment
the revision atomically. Read the returned record before the next edit.

```http
PUT /api/subscriptions/42
Content-Type: application/json

{"revision": 3, "name": "Updated name"}
```

Missing/zero revisions return HTTP 400 (`revision_required`). Stale revisions
return HTTP 409 (`revision_conflict`); backup destinations keep their existing
`backup_destination_changed` error. A missing resource may instead return 404.
Reload the record and review the intended change before retrying a conflict.
Blindly replacing the expected revision defeats stale-edit protection.

- `POST /api/subscriptions/:id/mark-renewed`: `{"revision": 3}`.
- `POST /api/subscriptions/batch`: add `"revisions": {"42": 3, "43": 7}`
  for every requested ID. Individual conflicts appear in the batch failures.
- Catalog reorder arrays: each item includes `id`, `sort_order`, and `revision`.
  A conflict rolls back the whole reorder.
- Icon upload: the supplied revision advances by one after success. Use this
  incremented revision when subsequently submitting the rest of the form.
- `GET /api/notifications/policy` returns revision 0 when no policy exists.
  The first `PUT` must explicitly supply `revision: 0`; an existing policy
  requires its positive revision. A successful first save returns revision 1.
- `GET /api/admin/settings` returns a `revisions` map for the complete settings
  snapshot. Send it back with `PUT /api/admin/settings`. All versions are checked
  within the same transaction as the settings writes; stale configuration causes
  the entire save to fail. `{}` is valid only for an empty settings table.

MCP `update_subscription`, `delete_subscription`, and
`mark_subscription_renewed` require both `revision` and `idempotency_key`.
Replay the original arguments/key to recover an already successful result. A
new operation using a stale revision fails without committing a success audit
event or idempotency record. After reviewing fresh data, use a new key.

This is a breaking change for clients that previously omitted preconditions.
The bundled web client and settings planning script send them automatically.
Existing rows migrate to revision 1; backup destination revisions are preserved.
Background lifecycle transitions compare the loaded revision and skip stale
rows. Database lock contention and cross-row limits remain separate concerns
from detecting stale row snapshots.
