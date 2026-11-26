# FatSecret MCP Server (Go) <img src="https://cdn.prod.website-files.com/6854cb4b7a1b8e39612d89cf/685543fddca41e9e9ce5ae5f_fatsecret_logo-op.png" alt="FatSecret Logo" height="28" />

[![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go)](go.mod)
[![MCP](https://img.shields.io/badge/MCP-2025--06--18-7C3AED)](https://modelcontextprotocol.io/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Tools](https://img.shields.io/badge/tools-56-brightgreen)](README.md#%EF%B8%8F-available-tools)

> [!IMPORTANT]
> This is **not an official MCP server** by FatSecret.
> It uses the [FatSecret Platform API](https://platform.fatsecret.com/) which requires a free developer account.

An MCP (Model Context Protocol) server that connects Claude/Cursor to the [FatSecret Platform API](https://platform.fatsecret.com/). Search foods, track your diet, manage recipes, and monitor weight directly from your AI assistant.

56 tools, config in `~/.fatsecret-mcp/config.json`.

Built on the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).

## ✨ Features

- 🔐 **Authentication** — Interactive credential setup and OAuth authorization
- ✋ **Draft → confirm** — Stage any write with `prepare_call` (validated + human-readable preview), execute with `confirm_draft`. Set `FATSECRET_REQUIRE_CONFIRM=1` to block direct writes entirely
- 🔍 **Food Search** — Search FatSecret's extensive food database with detailed nutrition data
- 📷 **Barcode Lookup** — Find foods by GTIN-13 barcode
- 🍳 **Recipe Search** — Browse and filter recipes by calories, macros, and prep time
- 📝 **Food Diary** — Add, edit, copy, and delete food diary entries
- 🍽️ **Saved Meals** — Create and manage reusable meal templates
- ⚖️ **Weight Tracking** — Record and view weight history
- 🏃‍♂️ **Exercise Tracking** — View exercises and manage activity entries
- ⭐ **Favorites** — Manage favorite foods and recipes

## 🏆 Why this server

Comparison as of November 2026. The official server is described in
[FatSecret's own docs](https://platform.fatsecret.com/docs/guides/mcp-server).

|                                  | This server                                      | Official `mcp.fatsecret.com` | Typical community servers  |
| -------------------------------- | ------------------------------------------------ | ---------------------------- | -------------------------- |
| Tools                            | 56                                               | 4                            | ~5–15                      |
| Diary / weight / exercise writes | ✅ full                                          | ❌ read-only                 | ⚠️ partial                 |
| Recipes, NLP, image recognition  | ✅                                               | ❌                           | ⚠️ rarely                  |
| Works on a free API key          | ✅ (v6 → v1 fallback)                            | ❌ Premier only              | depends                    |
| Self-hosted (binary / Docker)    | ✅                                               | ❌ hosted seats              | ✅                         |
| stdio + Streamable HTTP          | ✅                                               | remote only                  | usually stdio only         |
| Draft → confirm for writes       | ✅ (+ strict `FATSECRET_REQUIRE_CONFIRM=1` mode) | —                            | rarely                     |
| Self-contained binary            | ✅ ~14 MB, ~10 MB RAM, no runtime                | ❌ hosted seats              | ⚠️ usually needs a runtime |

Endpoints covered here that neither the official server nor — to our
knowledge — any community server exposes: `create_food_feedback`,
`get_food_classification_levels`, `commit_exercise_day`,
`create_profile` / `get_profile_auth`.

## 🚀 Quick Start

Build the binary (Go 1.27+):

```bash
go build -o fatsecret-mcp ./cmd/fatsecret-mcp
```

Or grab a ready binary / Docker image from
[Releases](https://github.com/kondratjev/fatsecret-mcp-go/releases):

```bash
docker run -i --rm \
  -e FATSECRET_CLIENT_ID=... \
  -e FATSECRET_CLIENT_SECRET=... \
  ghcr.io/kondratjev/fatsecret-mcp-go:latest
```

Add to your MCP client configuration:

```json
{
  "mcpServers": {
    "fatsecret": {
      "command": "/absolute/path/to/fatsecret-mcp"
    }
  }
}
```

That's it! On first use, the AI will guide you through setup:

1. **`check_auth_status`** — detects missing credentials and tells you what to do
2. **`setup_credentials`** — you provide your API keys (saved to `~/.fatsecret-mcp/config.json`)
3. **`start_auth`** → **`complete_auth`** — authorize your FatSecret account for diary/weight tools

Alternatively, you can pass credentials as environment variables:

```json
{
  "mcpServers": {
    "fatsecret": {
      "command": "/absolute/path/to/fatsecret-mcp",
      "env": {
        "FATSECRET_CLIENT_ID": "your_client_id",
        "FATSECRET_CLIENT_SECRET": "your_client_secret",
        "FATSECRET_CONSUMER_SECRET": "your_consumer_secret"
      }
    }
  }
}
```

### 🔑 Where to Get Credentials

1. Create a free account at [platform.fatsecret.com](https://platform.fatsecret.com/)
2. Navigate to **My Account → API Keys**
3. You'll see three values:
   - **Client ID** (used for both OAuth 2.0 and OAuth 1.0)
   - **Client Secret** (OAuth 2.0 - for public food/recipe search)
   - **Consumer Secret** (OAuth 1.0 - for user profile/diary access)

📖 Step-by-step guide: [Getting Started with FatSecret API](https://github.com/fatsecret-group/postman-fatsecret-apis)

#### 💎 Premier Keys

Premier keys work out of the box. FatSecret grants OAuth 2.0 scopes per key, so the server asks for `basic premier barcode nlp image-recognition localization feedback` and, if FatSecret refuses any of them, keeps just the ones your key is entitled to (free keys end up with `basic`). Premier keys get Premier-only data such as branded foods and `search_foods` v6; free keys get `search_foods` v1. Barcode lookup needs the `barcode` scope; `parse_food_text` and `recognize_food_image` need the paid NLP and image recognition add-ons.

### Claude Desktop (Manual)

`~/Library/Application Support/Claude/claude_desktop_config.json` — same JSON as above.

### Claude Code (CLI)

```bash
claude mcp add fatsecret -- /absolute/path/to/fatsecret-mcp
```

Or with env vars:

```bash
claude mcp add fatsecret \
  -e FATSECRET_CLIENT_ID=your_client_id \
  -e FATSECRET_CLIENT_SECRET=your_client_secret \
  -e FATSECRET_CONSUMER_SECRET=your_consumer_secret \
  -- /absolute/path/to/fatsecret-mcp
```

Verify with `claude mcp list`.

### Cursor

- **Settings UI** — `Settings → MCP → + Add new MCP server`, then fill in the command and env
- **Project config** — add JSON to `.cursor/mcp.json` in your project root
- **Global config** — add JSON to `~/.cursor/mcp.json`

### ChatGPT

Remote only — paste your self-hosted URL (`Settings → Connectors → Add`):

```
https://<DOMAIN>/<MCP_SECRET>/mcp
```

### OpenCode

`~/.config/opencode/opencode.json` — note the key is `mcp`, env is `environment`:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "fatsecret": {
      "type": "local",
      "command": ["/absolute/path/to/fatsecret-mcp"],
      "environment": {
        "FATSECRET_CLIENT_ID": "your_client_id",
        "FATSECRET_CLIENT_SECRET": "your_client_secret",
        "FATSECRET_CONSUMER_SECRET": "your_consumer_secret"
      },
      "enabled": true
    }
  }
}
```

Or point at your self-hosted server instead of a local binary:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "fatsecret": {
      "type": "remote",
      "url": "https://<DOMAIN>/<MCP_SECRET>/mcp",
      "enabled": true
    }
  }
}
```

## 🛠️ Available Tools

| Tool                                | Description                                                                |
| ----------------------------------- | -------------------------------------------------------------------------- |
| **🔐 Setup & Auth**                 |                                                                            |
| `check_auth_status`                 | Check if credentials and profile auth are configured. **Call this first.** |
| `setup_credentials`                 | Save FatSecret API credentials to persistent config                        |
| `start_auth`                        | Start OAuth 1.0 authorization — returns URL for user to visit              |
| `complete_auth`                     | Complete OAuth with verifier PIN from authorization page                   |
| `prepare_call`                      | Stage a write as a single-use draft (validated + preview, nothing written) |
| `confirm_draft`                     | Execute a draft staged by `prepare_call`                                   |
|                                     |                                                                            |
| **🔍 Food Search** _(public)_       |                                                                            |
| `search_foods`                      | Search the food database                                                   |
| `get_food`                          | Get detailed nutrition info for a food                                     |
| `find_food_by_barcode`              | Find food by GTIN-13 barcode                                               |
| `autocomplete_foods`                | Get search autocomplete suggestions                                        |
|                                     |                                                                            |
| **🤖 Recognition** _(paid add-ons)_ |                                                                            |
| `parse_food_text`                   | Match a free-text meal description to foods and servings (NLP)             |
| `recognize_food_image`              | Recognize foods in a meal photo (image recognition)                        |
| `create_food_feedback`              | Report food/barcode data issues, get upload URLs (paid feedback add-on)    |
|                                     |                                                                            |
| **🍳 Recipes** _(public)_           |                                                                            |
| `search_recipes`                    | Search recipes with filters                                                |
| `get_recipe`                        | Get recipe details with ingredients and directions                         |
|                                     |                                                                            |
| **📚 Reference** _(public)_         |                                                                            |
| `get_food_categories`               | Get food categories                                                        |
| `get_food_sub_categories`           | Get food sub categories                                                    |
| `get_food_classification_levels`    | Get food classification hierarchy (Premier)                                |
| `get_brands`                        | Get food brands                                                            |
| `get_recipe_types`                  | Get recipe types                                                           |
|                                     |                                                                            |
| **📝 Food Diary** _(profile auth)_  |                                                                            |
| `log_meal`                          | Log a whole meal in one call (grams or servings) and get day totals        |
| `get_food_entries`                  | Get food diary entries for a date                                          |
| `get_food_entries_month`            | Get monthly nutrition summary (calories & macros per day)                  |
| `create_food_entry`                 | Add a food diary entry                                                     |
| `edit_food_entry`                   | Edit a food diary entry                                                    |
| `delete_food_entry`                 | Delete a food diary entry                                                  |
| `copy_food_entries`                 | Copy entries from one date to another                                      |
| `copy_saved_meal_entries`           | Copy a saved meal to a date                                                |
|                                     |                                                                            |
| **⭐ Favorites** _(profile auth)_   |                                                                            |
| `get_favorite_foods`                | Get favorite foods                                                         |
| `add_favorite_food`                 | Add food to favorites                                                      |
| `delete_favorite_food`              | Remove food from favorites                                                 |
| `get_most_eaten_foods`              | Get most eaten foods                                                       |
| `get_recently_eaten_foods`          | Get recently eaten foods                                                   |
| `get_favorite_recipes`              | Get favorite recipes                                                       |
| `add_favorite_recipe`               | Add recipe to favorites                                                    |
| `delete_favorite_recipe`            | Remove recipe from favorites                                               |
|                                     |                                                                            |
| **🍽️ Saved Meals** _(profile auth)_ |                                                                            |
| `get_saved_meals`                   | Get saved meals                                                            |
| `create_saved_meal`                 | Create a saved meal                                                        |
| `edit_saved_meal`                   | Edit a saved meal                                                          |
| `delete_saved_meal`                 | Delete a saved meal                                                        |
| `get_saved_meal_items`              | Get items in a saved meal                                                  |
| `add_saved_meal_item`               | Add food to a saved meal                                                   |
| `edit_saved_meal_item`              | Edit saved meal item                                                       |
| `delete_saved_meal_item`            | Remove item from saved meal                                                |
|                                     |                                                                            |
| **⚖️ Weight** _(profile auth)_      |                                                                            |
| `update_weight`                     | Record weight for a date                                                   |
| `get_weight_month`                  | Get weight history for a month                                             |
|                                     |                                                                            |
| **🏃‍♂️ Exercise** _(profile auth)_    |                                                                            |
| `get_exercises`                     | Get exercise types                                                         |
| `get_exercise_entries`              | Get exercise entries for a date                                            |
| `edit_exercise_entries`             | Shift exercise time between activities                                     |
| `get_exercise_entries_month`        | Get exercise data for a month                                              |
| `save_exercise_template`            | Save exercise template for weekdays                                        |
| `commit_exercise_day`               | Commit template entries for a date                                         |
|                                     |                                                                            |
| **👤 Profile** _(profile auth)_     |                                                                            |
| `get_profile`                       | Get user profile info                                                      |
| `create_food`                       | Create a custom food (Premier)                                             |
| `create_profile`                    | Create a detached profile (returns tokens, no switch)                      |
| `get_profile_auth`                  | Get auth tokens for a detached profile                                     |

<sub>API reference: [FatSecret Postman Collection](https://www.postman.com/fatsecret/fatsecret-public-apis/)</sub>

## 🧪 Test Connection

List tools over stdio (keep stdin open — the server exits on EOF):

```bash
{ printf '%s\n' \
 '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}' \
 '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
 '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'; sleep 4; } | ./fatsecret-mcp
```

Or serve HTTP and probe it:

```bash
PORT=8000 ./fatsecret-mcp &
curl localhost:8000/healthz  # → ok
```

With credentials:

```bash
FATSECRET_CLIENT_ID='...' FATSECRET_CLIENT_SECRET='...' FATSECRET_CONSUMER_SECRET='...' ./fatsecret-mcp
```

## 📋 Requirements

- Go 1.27+ to build (or just the binary — no runtime needed)
- FatSecret Platform API account ([platform.fatsecret.com](https://platform.fatsecret.com/))
- MCP-compatible client (Claude Desktop, Cursor, etc.)

## 🌍 Self-Hosted Server (Docker)

Run the server on your own machine or VPS and connect remote clients (Claude.ai, ChatGPT connectors) over HTTPS.

1. Copy `.env.example` to `.env` and fill it in:
   - `FATSECRET_CLIENT_ID`, `FATSECRET_CLIENT_SECRET`, `FATSECRET_CONSUMER_SECRET`
   - `TZ` — time zone for the default diary "today" (e.g. `Europe/Moscow`)
   - `DOMAIN` — public host name (an IP works too, e.g. `203-0-113-5.sslip.io`)
   - `MCP_SECRET` — long random string: `openssl rand -hex 24`
2. `docker compose up -d --build`

Caddy issues HTTPS automatically and proxies only `https://<DOMAIN>/<MCP_SECRET>/mcp` to the server — everything else gets 404. The secret path is the access control, since remote MCP connectors can't send auth headers. Guard the URL like a password. The image is a static Go binary serving stateless Streamable HTTP natively (no gateway, no per-session processes), so memory stays flat no matter how many prompts hit it.

Profile tokens persist in `./data`, so `setup_credentials` and favorites survive restarts and rebuilds.

### 🚀 Deploy to VPS

Single source of truth is this repo — never edit code on the server. The server copy is a clean mirror (`git reset --hard`; `.env` and `./data` are git-ignored and survive).

```sh
VPS_SSH=user@host VPS_DIR=/opt/fatsecret-mcp-go ./scripts/deploy.sh
```

This pushes the current branch, then on the server pulls it, rebuilds and restarts (`docker compose`, or legacy `docker-compose` if v2 is missing). Export `VPS_SSH`/`VPS_DIR` in your shell rc to run bare `./scripts/deploy.sh`. On a tiny VPS: `docker builder prune -af` if a build runs out of disk, add swap if the build OOMs.

## 🔧 Development

```
.
├── cmd/fatsecret-mcp/  # entrypoint: stdio vs stateless HTTP, tool registration
├── internal/
│   ├── api/            # Server state, OAuth2 token cache, public/profile requests
│   ├── auth/           # OAuth 2.0 (scopes, detection) + OAuth 1.0 (HMAC-SHA1)
│   ├── config/         # ~/.fatsecret-mcp/config.json + env overrides
│   ├── httpx/          # resilientFetch: timeouts + GET-only retries
│   ├── platform/       # dates, servings, scope-14 detection, search trimming
│   └── tools/          # 56 tool definitions + handlers
├── Dockerfile
└── docker-compose.yml  # mcp + Caddy (HTTPS, secret path)
```

```bash
go build ./...   # build
go vet ./...     # lint
go test ./...    # tests (OAuth1 RFC 5849 vector, dates, servings)
```

Debugging with MCP Inspector:

```bash
FATSECRET_CLIENT_ID=X FATSECRET_CLIENT_SECRET=X FATSECRET_CONSUMER_SECRET=X npx -y @modelcontextprotocol/inspector /absolute/path/to/fatsecret-mcp
```

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.
