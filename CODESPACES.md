# Run ARTEX in GitHub Codespaces

This repository includes a private-by-default GitHub Codespaces development environment.
It runs the ARTEX source code, PostgreSQL, the Go API/MCP server, and the Next.js UI.

## github.dev vs GitHub Codespaces

- **github.dev** is the browser-based editor for editing and committing files. It does not provide a terminal or compute to run ARTEX or PostgreSQL.
- **GitHub Codespaces** provisions a development VM and builds the environment described in `.devcontainer/`. Use Codespaces when you want to run the application.

See the [GitHub documentation about github.dev and Codespaces](https://docs.github.com/en/codespaces/the-githubdev-web-based-editor).

## Start a Codespace

1. Open this repository on GitHub.
2. Select **Code → Codespaces → Create codespace on main**. To enter LLM keys at creation, use **New with options** and provide the recommended secrets.
3. Wait for the dev container to finish building and for its post-create setup to complete.
4. The post-start script starts the Go backend and Next.js frontend automatically.
5. Open the **Ports** tab and follow the private **ARTEX web UI** port (5173).

The backend/API is on port 8787. The MCP endpoint, after it is enabled in ARTEX, is:

`https://<your-codespace-name>-8787.app.github.dev/mcp`

Codespaces forwards both application ports as **private**. Do not change them to public for convenience.

## First-time ARTEX setup

1. Open the forwarded port 5173 and create the first administrator password.
2. Configure an LLM provider in ARTEX settings before starting AI-assisted exploration.
3. Under **System Settings → System & Accounts → MCP Server**, enable the MCP server.
4. Create a separate MCP token. Begin with the `read` scope.
5. Store the token in a secret manager or the local MCP client's protected configuration. It is shown only once.

The MCP endpoint is served by the backend on port 8787. Do not commit access tokens, LLM API keys, database passwords, `config.json`, `jwt.key`, or runtime data to Git.

## Database and environment

- PostgreSQL 16 runs as a separate Compose service.
- Its port is not published to the Codespaces host.
- ARTEX connects to it over the private Compose network using `ARTEX_PG_DSN`.
- The database uses a disposable development-only password on the private Compose network. Its port is not published. Never reuse this credential or this setup for production.
- Optional `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, and `GEMINI_API_KEY` values can be configured as Codespaces secrets. The startup script passes available values to ARTEX.
- Codespaces secrets are injected after the environment is created, so they are not used for Docker image or Compose build-time configuration.
- The database volume persists across stops of the same Codespace. It is not a production backup.

## Useful commands

Start the services again:

```bash
bash .devcontainer/start-services.sh
```

Check the backend:

```bash
curl --fail http://127.0.0.1:8787/api/health
```

View logs:

```bash
tail -f .codespaces-backend.log
tail -f .codespaces-frontend.log
```

Stop the development services:

```bash
if [ -f .codespaces-backend.pid ]; then kill "$(cat .codespaces-backend.pid)" 2>/dev/null || true; fi
if [ -f .codespaces-frontend.pid ]; then kill "$(cat .codespaces-frontend.pid)" 2>/dev/null || true; fi
```

## Connecting ChatGPT

A private Codespaces port is protected by GitHub authentication. A remote ChatGPT service cannot automatically use your interactive GitHub session cookie. Keep the port private while developing.

For ChatGPT integration, connect through a local MCP client/bridge that you control, or later deploy ARTEX behind a dedicated HTTPS endpoint with a scoped Bearer token and source restrictions. Never commit or paste the MCP token into this repository.

## GitHub Actions

The CI workflow runs Go tests and frontend checks for pushes and pull requests to `main`. It validates the codebase; it does not provide a continuously running ARTEX server. Production deployment requires a separate deployment target and protected environment secrets.
