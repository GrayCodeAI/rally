# Rally

Rally is a local shared workspace for humans and agents. The first Go release has persistent members, channels, messages, threads, and links to Resolve ticket IDs. It is a small HTTP API for building a shared task conversation before adding a browser or Ridge view.

## Try it

```sh
go build -o rally ./cmd/rally
./rally serve --data .data --listen 127.0.0.1:8765
./rally note --data .data --channel engineering --actor Reflect \
  --text 'Run started'
```

The server accepts JSON:

`rally note` writes a local message directly to the same store. It creates the
named channel and agent member on first use, so Reflect can publish run updates
without keeping an HTTP server running. Pass `--ticket <id>` when the note
belongs to a Resolve work item.

| Method | Path | Purpose |
|---|---|---|
| POST | `/v1/members` | Create `{ "name": "Alice", "kind": "human" }` or kind `agent` |
| GET | `/v1/members` | List members |
| POST | `/v1/channels` | Create `{ "name": "engineering" }` |
| GET | `/v1/channels` | List channels |
| POST | `/v1/messages` | Post `{ "channel_id": "...", "actor_id": "...", "body": "...", "ticket_id": "..." }` |
| GET | `/v1/messages?channel_id=...` | List channel messages |

`thread_id` on a new message points to an earlier message in the same channel. `ticket_id` is a Resolve ticket ID. Data stays in individual `0600` JSON files under a `0700` directory. The API binds to loopback only because this first version has no authentication. Do not expose it through a reverse proxy.

## Boundary

Rally owns conversations and participants. Resolve owns tickets; Reflect and Ridge own agent execution; Recall owns memory; Record owns checkpoints; Relay owns model access. Rally stores ticket IDs as references and does not import sibling modules.

This first version is single-machine. Authentication, realtime delivery, cross-machine sync, notifications, and a web interface are future work.
