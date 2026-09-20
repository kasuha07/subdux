# Login identifiers

Password login trims surrounding whitespace, then looks up the identifier as an
email address (case-insensitive). Only when no email matches does it look up an
exact, case-sensitive username. A wrong password or disabled account never causes
a fallback to another account.

Registration, administrator-created accounts, OIDC account creation, and email
changes reject identifiers that collide with another account's opposite field.
These checks also run inside the write transaction.

Existing cross-field collisions do not require a destructive migration: the email
owner can sign in with that email, and the account whose username collides can
sign in using its own email. No account is renamed or merged automatically.
