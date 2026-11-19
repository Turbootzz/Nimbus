# Connect AI assistants (MCP)

Nimbus has a [Model Context Protocol](https://modelcontextprotocol.io) endpoint at `/mcp`. Connect Claude, Open WebUI or another MCP client to it and ask things like "what is down?" or "how was Jellyfin's uptime this week?". The assistant reads the answer from your dashboard.

Nimbus holds no LLM keys and calls no AI service. It only answers the requests your assistant sends, and only with a token you created.

## What an assistant can do

| Tool | What it returns |
|---|---|
| `list_services` | All services with their status, optionally for one group |
| `get_service_status` | One service (by ID or name) with its latest health checks and their errors |
| `get_uptime` | Uptime percentage and response times over `1h`, `24h`, `7d` or `30d` |
| `search_services` | Services whose name, description, URL or group contains a text |
| `get_widget_data` | Settings and latest data of one widget |
| `list_integrations` | Connected apps and whether their connection works. No URLs, no credentials |
| `recheck_service` | Runs a health check now. The only tool that changes anything, see below |

There is also one resource, `nimbus://dashboard`: a short text summary with the status counts, the offline services, the groups, the widgets and the integrations.

Everything is limited to the user who owns the token. An assistant sees no more than that user sees in Nimbus, with the same masking of secrets, and nothing of other users.

## 1. Create a token

1. Open **Settings → API Tokens**.
2. Give the token a name, for example `Claude`.
3. Select the **MCP** scope. You can clear the other scopes: `mcp` is enough for every read tool.
4. Optional: select **Write services** (`write:services`) too if the assistant may run `recheck_service`.
5. Optional: choose when the token expires.
6. Select **Create Token** and copy the token. It starts with `nimbus_` and is shown once.

See [API Tokens](CONFIGURATION.md#api-tokens) for all scopes, expiry and the rate limit.

## 2. Find the URL

| Setup | URL |
|---|---|
| Docker image (default) | `http://<host>:3000/mcp` |
| Behind your own reverse proxy with HTTPS | `https://nimbus.example.com/mcp` |
| Separate containers (legacy) | `http://<host>:8080/mcp` (the backend) |
| Local development | `http://localhost:8080/mcp` (the backend port) |

A reverse proxy that forwards everything to Nimbus needs no extra rule. The endpoint answers each request with one JSON response, so there is no long-lived connection to keep open.

Check that it works:

```bash
curl -s http://localhost:3000/mcp \
  -H "Authorization: Bearer nimbus_your_token" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_services","arguments":{}}}'
```

## 3. Connect a client

The examples use `http://localhost:3000/mcp` and `nimbus_your_token`. Replace both.

Only the `curl` check above has been run against a real Nimbus. The Claude Code and Claude Desktop (`mcp-remote`) snippets follow the documentation of those tools as of November 2026 but are **untested** end to end, and so are the Open WebUI and Home Assistant setups. Please open an issue if a step is off.

### Claude Code

```bash
claude mcp add --transport http nimbus http://localhost:3000/mcp \
  --header "Authorization: Bearer nimbus_your_token"
```

Add `--scope user` to have it in every project. Run `/mcp` in Claude Code to check the connection.

To share the server with a team without sharing a token, commit a `.mcp.json` that reads the token from each person's environment:

```json
{
  "mcpServers": {
    "nimbus": {
      "type": "http",
      "url": "http://localhost:3000/mcp",
      "headers": {
        "Authorization": "Bearer ${NIMBUS_TOKEN}"
      }
    }
  }
}
```

### Claude Desktop

`claude_desktop_config.json` only starts local programs, so it needs a small bridge to reach a URL: [`mcp-remote`](https://github.com/geelen/mcp-remote), which runs with Node.js 18 or newer.

Open the Claude menu, then **Settings… → Developer → Edit Config**. That opens the config file (macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`, Windows: `%APPDATA%\Claude\claude_desktop_config.json`). Add:

```json
{
  "mcpServers": {
    "nimbus": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-remote",
        "http://localhost:3000/mcp",
        "--allow-http",
        "--header",
        "Authorization:${NIMBUS_AUTH}"
      ],
      "env": {
        "NIMBUS_AUTH": "Bearer nimbus_your_token"
      }
    }
  }
}
```

- Quit Claude Desktop completely and start it again. The config is only read at startup.
- `--allow-http` lets `mcp-remote` use a plain `http://` URL. Only use that on a network you trust, and remove the line when the URL is `https://`.
- The header has no space after the colon on purpose: Claude Desktop on Windows breaks arguments that contain spaces. The space lives in `NIMBUS_AUTH` instead.

Custom connectors in Claude are a different mechanism from this config file. As far as we know they connect from Anthropic's servers and not from your computer, so they do not fit a Nimbus on your LAN (untested).

### Open WebUI

Open WebUI 0.6.31 or newer talks to the endpoint directly. Only an administrator can add it.

1. Open **Settings → Integrations** in the admin section (older versions call it **Admin Settings → External Tools** or **Tool Servers**).
2. Under **External Tool Servers**, select **+ Add Connection**.
3. Set **Type** to **MCP (Streamable HTTP)**. Do not leave it on OpenAPI.
4. Enter the URL. From an Open WebUI container, `localhost` is the container itself: use the address of the Nimbus host or its container name, for example `http://nimbus:3000/mcp`.
5. Set **Auth** to **Bearer** and paste the token as the key (the name of this option is not confirmed against the Open WebUI documentation).
6. Save. Restart Open WebUI if it asks for it.

### Home Assistant

The [Model Context Protocol integration](https://www.home-assistant.io/integrations/mcp/) of Home Assistant can use the Nimbus tools in Assist. It signs in with OAuth only and has no field for a token, and Nimbus has no OAuth sign-in for MCP yet. Until then, put a proxy in between that adds the token.

With [Caddy](https://caddyserver.com), next to Nimbus:

```caddyfile
:8787 {
	handle /mcp {
		reverse_proxy nimbus:3000 {
			header_up Authorization "Bearer nimbus_your_token"
		}
	}
	handle {
		respond 404
	}
}
```

Then add the integration in Home Assistant with the URL `http://<caddy-host>:8787/mcp`. This proxy setup is untested: it relies on Home Assistant skipping its OAuth sign-in when the server does not ask for one.

Anyone who can reach that port can use the token, without knowing it. Only expose the port to Home Assistant, and use a token with just the `mcp` scope.

Home Assistant uses tools only, so the `nimbus://dashboard` resource is not available there.

### Other clients

Any client that supports the **Streamable HTTP** transport and can send a header works:

- URL: `http://<host>:3000/mcp`
- Header: `Authorization: Bearer nimbus_your_token`

Clients that only start local programs (stdio) can use `mcp-remote` as in the Claude Desktop example. The endpoint does not offer the older SSE transport.

## Example prompts

- "What is down right now?"
- "How was Jellyfin's uptime this week?"
- "Which of my services are slowest?"
- "Is anything in the Media group offline?"
- "Why is Sonarr failing its health check?"
- "Are all my integrations still connected?"
- "What does my weather widget show?"
- "Check Pi-hole again." (needs `write:services`)

## Security notes

- **Read-only by default.** A token with only the `mcp` scope can read and nothing else. `recheck_service` needs `write:services` on the same token, and all it does is run a health check.
- **`mcp` is not API access.** The scope opens `/mcp` only. To use the REST API the token needs the other scopes.
- **No secrets.** Credentials of integrations are never returned, and neither are the URLs or options of integrations. Secrets in widget settings are masked like they are in the web UI, and URLs in widget settings lose their query string and user info, where keys often live.
- **The assistant sees your dashboard.** Service names, URLs, descriptions and widget content go to the AI provider of the client you connect. Use a local model if that matters to you.
- **Text from your apps is data.** Widget content such as RSS items or calendar entries comes from outside. Nimbus tells clients to treat it as data, but be careful with assistants that can also act on other tools.
- **Keep it on your network.** `/mcp` is as reachable as Nimbus itself. On a LAN that is fine. If Nimbus is reachable from the internet, use HTTPS: a token sent over plain HTTP can be read on the way.
- **Reverse proxy.** `/mcp` needs the `Authorization` header to arrive unchanged. A proxy that does its own sign-in (Authelia, Authentik, basic auth) in front of `/mcp` blocks MCP clients; exclude that path there. The token still protects it.
- **One token per client.** Then you can revoke a single client under **Settings → API Tokens**, and "last used" shows which ones are in use. Set an expiry for tokens you hand out for a short time.
- **Rate limit.** Each token may make `API_TOKEN_RATE_LIMIT` requests per minute (default 600). Above that the endpoint answers `429`.

## Troubleshooting

| Answer | Cause |
|---|---|
| `401` | No token, a wrong, revoked or expired token, or the header did not arrive (check your reverse proxy) |
| `403` | The token has no `mcp` scope. Create a new token with it; scopes of an existing token cannot be changed |
| `404` or a web page | The request reached the frontend. Check that the path is exactly `/mcp` and, with separate containers, that the URL points at the backend |
| `413` | The request is larger than 1 MB |
| `405` | The client sent `GET`. That is normal for clients probing for a stream; they continue with `POST` |
| `429` | Rate limit. Wait a minute or raise `API_TOKEN_RATE_LIMIT` |
| "this API token is read-only" | `recheck_service` needs a token that also has `write:services` |
