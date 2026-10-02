# MCP OAuth

Subdux supports browser-based OAuth authorization for remote MCP clients alongside existing MCP-client API keys. OAuth access and refresh tokens are separate from human REST sessions and can only authorize `/mcp` tools.

## Configure the instance

1. Enable MCP in administrator settings.
2. Set **Site URL** to the public HTTPS origin, for example `https://subdux.example.com`. OAuth uses this configured origin for discovery, issuer identification and token resource binding. Paths, queries, fragments and embedded credentials are not supported. HTTP is accepted only for `localhost`, `127.0.0.1` and `[::1]` development instances.
3. Ensure clients can reach the MCP and OAuth endpoints. Hosted ChatGPT and Claude connections originate outside your local network. Keep reverse-proxy authentication from intercepting discovery and OAuth requests; Subdux verifies the application credentials itself.

Existing API-key connections continue working without a configured Site URL.

## Connect a client

Add `https://subdux.example.com/mcp` as a Streamable HTTP server. Select OAuth when the client offers an authentication choice. Open its sign-in flow, log in to Subdux using any existing supported method, and approve the displayed client and permissions. No Subdux password or human-session token is shared with the MCP client.

Clients can identify themselves through CIMD (an HTTPS Client ID Metadata Document), DCR (`POST /oauth/register`), or a previously registered client ID. The supported token endpoint authentication method is `none`: public clients use mandatory PKCE `S256`, without a client secret. Client credentials, implicit grants and password grants are not supported. Client metadata names are supplied by clients and do not prove publisher identity.

The authorization-code flow requires `response_type=code`, `client_id`, `redirect_uri`, `code_challenge`, `code_challenge_method=S256`, and `resource=https://subdux.example.com/mcp`. The token request also requires that same `resource`. Redirect URIs match registration exactly, with the RFC 8252 exception for the port of an HTTP loopback URI. Authorization responses include `iss` and echo the client's `state`.

Available scopes:

| Scope | Permission |
| --- | --- |
| `read` | Read subscription tools and reference data; default permission |
| `write` | Create, update, renew and delete subscriptions; includes read |
| `offline_access` | Optional OAuth scope indicating refresh-token use |

The resource metadata advertises the minimal `read` scope. A client that needs writing should request `read write` (or use the tool's insufficient-scope challenge to authorize again). The consent page leaves writing unchecked, even when requested. Refresh requests may narrow permissions but cannot expand them.

In **Settings → API keys → MCP OAuth connections**, copy the configured endpoint, inspect connections and revoke access. Each grant lasts at most 30 days; access tokens last at most 15 minutes. Refresh tokens rotate on every exchange. Reusing an already consumed authorization code or refresh token revokes its token family. Clients must serialize refresh requests and securely persist the replacement refresh token. Reconnect after grant expiry or revocation.

## Endpoints

| Endpoint | Purpose |
| --- | --- |
| `GET /.well-known/oauth-protected-resource/mcp` | Resource discovery; root well-known alias also available |
| `GET /.well-known/oauth-authorization-server` | Issuer, endpoints, scopes and capabilities |
| `GET /oauth/authorize` | Start the browser authorization interaction |
| `POST /oauth/register` | JSON public-client registration |
| `POST /oauth/token` | Form-encoded code exchange or refresh |
| `POST /oauth/revoke` | Form-encoded RFC 7009 token revocation |
| `GET /api/mcp/oauth/info` | Human-session connection configuration |
| `GET/POST /api/mcp/oauth/requests/:request` | Human-session consent review and decision |
| `GET /api/mcp/oauth/grants` | Human-session authorized connections |
| `DELETE /api/mcp/oauth/grants/:id` | Human-session revocation |

Unauthorized MCP requests return a Bearer challenge with the resource metadata URL. Requests with an `Authorization` header must pass OAuth validation; an invalid Bearer token never falls back to an API key. Access tokens are validated for the current resource, grant lifetime, revocation, scope and active user status on every request. They never grant account, export or administrator access.

OAuth clients, authorization interactions, token hashes and grants are persisted in the instance database. Write audit and idempotency records include the stable OAuth grant/client identity. Existing per-user idempotency semantics and transactional mutation/audit/result persistence are preserved.

CIMD fetching uses strict public-network egress, rejects redirects, bounds response size and timeout, and caches metadata for at most one hour. Registration and authorization requests have body, rate and storage limits. Credentials and interaction handles are redacted from Subdux request logs; reverse proxies should also redact query strings for OAuth and consent routes.

## Validation

The repository tests exercise the full HTTP code/PKCE flow, bearer discovery, public registration, client metadata validation, read/write separation, REST rejection, revocation, replay, concurrent exchanges, audit attribution and schema upgrades. Run `make check` and `cd web && bun run test`.

Real ChatGPT, Claude and Codex account linking still requires verification against the deployed HTTPS instance. Local tests do not establish hosted-client interoperability or account/workspace availability.
