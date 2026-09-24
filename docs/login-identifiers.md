# Login identifiers

Password login trims surrounding whitespace, then looks up the identifier as an
email address (case-insensitive). Only when no email matches does it look up an
exact, case-sensitive username. A wrong password or disabled account never causes
a fallback to another account.

Registration, administrator-created accounts, OIDC account creation, and email
changes reject identifiers that collide with another account's opposite field.
These checks also run inside the write transaction.

PostgreSQL additionally enforces case-insensitive email uniqueness with an
index on `lower(email)`. A user-table trigger serializes claims for email and
username identifiers and rejects new cross-field collisions, including those
from concurrent requests. Existing cross-field collisions remain unchanged.
If existing emails differ only by case, migration stops and reports that the
duplicate accounts must be resolved manually before the index can be created.
SQLite continues to use the service-level checks.

Existing cross-field collisions do not require a destructive migration: the email
owner can sign in with that email, and the account whose username collides can
sign in using its own email. No account is renamed or merged automatically.
