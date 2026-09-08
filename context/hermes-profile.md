# Restricted Hermes Verdict Profile

This runbook creates a **separate local profile** for structured market-evidence
opinions. It has no wallet, signing, transaction, filesystem, terminal,
browser, memory, delegation, cron, messaging, or tool authority.

> Do not clone the default profile: cloning can copy credentials, skills, and
> state into this restricted boundary. Do not copy any existing Hermes or
> TwitterAPI.io credential into this profile.

## Create the empty profile

```bash
hermes profile create hypeverdict --no-skills --no-alias \
  --description "Returns schema-validated paper-trading evidence verdicts only."
```

Use `hermes profile show hypeverdict` to confirm it exists. Use the profile
flag for every later command; this avoids changing the operator's sticky
default profile:

```bash
hermes -p hypeverdict config path
hermes -p hypeverdict config env-path
```

## Configure API-server credentials locally

Generate a new bearer value outside this repository and place it only in the
restricted profile's ignored `.env` file. Never commit, print, or paste the
value into chat, source, logs, or Hermes instructions.

Required environment names:

```dotenv
API_SERVER_ENABLED=true
API_SERVER_PORT=8642
API_SERVER_KEY=<operator-generated-local-secret>
```

The API server binds to loopback by default. Do not add CORS origins: this Go
service calls Hermes server-to-server and browser access is prohibited.

## Restrict tools

Start the restricted gateway with the profile-scoped command below. The verdict
client must send a single stateless `/v1/responses` request and must never
include a Hermes session header, conversation, or previous response ID.

Disable every available API-server toolset with the profile-scoped CLI:

```bash
hermes -p hypeverdict tools disable --platform api_server \
  web browser terminal file code_execution vision video image_gen video_gen \
  bfl x_search tts stt skills todo memory context_engine session_search \
  clarify delegation cronjob homeassistant spotify yuanbao computer_use a2a
```

Do not enable MCP servers, plugins, computer use, voice, or messaging platforms
for this profile. Verify the resolved `api_server` tools after startup:

```bash
curl -H "Authorization: Bearer $API_SERVER_KEY" \
  http://127.0.0.1:8642/v1/toolsets
```

## Start and verify

Start the profile's gateway only after the model/provider configuration and
new bearer credential are present locally:

```bash
hermes -p hypeverdict gateway run
```

Verify liveness without credentials:

```bash
curl http://127.0.0.1:8642/health
```

Verify authenticated readiness using the local bearer value without printing it:

```bash
curl -H "Authorization: Bearer $API_SERVER_KEY" \
  http://127.0.0.1:8642/health/detailed
```

Expected behavior:

- `/health` returns a successful liveness response.
- `/health/detailed` returns a successful readiness response.
- no tool call, file access, terminal access, browser access, persistent
  memory, delegation, cron, or messaging capability is available.
- the service is reachable only at `127.0.0.1:8642`.

## Structured verdict compatibility

The local API server accepts an OpenAI-compatible `response_format` field, but
the current Codex route does not enforce that JSON schema in a Responses API
request. The Go adapter therefore sends both the compatibility schema hint and
an explicit bounded verdict JSON contract in its stateless instruction. Its
local `domain.Verdict` validation remains the authority: malformed, unsupported,
out-of-range, or unknown-evidence outputs fail closed and are never persisted
or used for paper admission.
