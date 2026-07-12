# statuswatch

A lightweight, self-hosted status-page monitor that sends Discord notifications
when third-party services report incidents, publish updates, recover, or change
their overall health.

## Overview

statuswatch polls the official status pages of the services you configure. It
currently supports Atlassian Statuspage and Status.io, including optional
component filters for Atlassian-powered pages and a global minimum incident
impact threshold.

The monitor stores incident and service-health state beside the executable. This
allows it to distinguish new activity from previously reported events and avoids
sending the same alert on every poll. It is distributed as a single Go binary
and is designed to run continuously on a server or VPS.

## Features

- Atlassian Statuspage and Status.io provider support
- Discord embeds for new incidents, incident updates, resolutions, degraded
  services, and recoveries
- Per-service component filtering for Atlassian-powered status pages
- Configurable minimum incident impact: `none`, `minor`, `major`, or `critical`
- Persistent state to prevent duplicate notifications across restarts
- Immediate status check at startup, followed by checks on a configurable interval
- HTTP timeouts and error logging when a provider or Discord cannot be reached
- A single self-contained binary with no runtime dependencies
- Straightforward cross-compilation for Linux, Windows, and other Go targets

## Requirements

- [Go](https://go.dev/) 1.26.2 or newer to build from source
- A Discord webhook URL
- Network access to Discord and every configured status page

Go is not required on the deployment server when you copy a compiled binary to
it.

## Setup

Clone the repository and build the executable:

```bash
git clone <repository-url>
cd statuswatch
go build -o statuswatch ./src
```

On Windows, use an `.exe` output name:

```powershell
go build -o statuswatch.exe ./src
```

Copy the example configuration next to the compiled executable:

```bash
cp config.example.json config.json
```

On Windows:

```powershell
Copy-Item config.example.json config.json
```

Edit `config.json`, add your Discord webhook, and configure the services you
want to monitor. Then start statuswatch:

```bash
./statuswatch
```

```powershell
.\statuswatch.exe
```

statuswatch performs its first check immediately and logs each result to
standard output.

> [!IMPORTANT]
> `config.json` must be in the same directory as the executable, regardless of
> the directory from which statuswatch is launched. Keep it private because it
> contains your Discord webhook URL.

## Configuration

```json
{
  "webhook_url": "YOUR_DISCORD_WEBHOOK_URL_HERE",
  "poll_interval_seconds": 120,
  "min_impact": "major",
  "services": [
    {
      "name": "Roblox",
      "url": "https://api.status.io/1.0/status/59db90dbcdeb2f04dadcf16d",
      "provider": "statusio"
    },
    {
      "name": "Discord",
      "url": "https://discordstatus.com",
      "provider": "atlassian"
    },
    {
      "name": "GitHub",
      "url": "https://www.githubstatus.com",
      "provider": "atlassian",
      "components": ["Git Operations", "API Requests", "Actions"]
    }
  ]
}
```

### Top-level options

| Option | Required | Description |
| --- | --- | --- |
| `webhook_url` | Yes | Discord webhook that receives status embeds. |
| `poll_interval_seconds` | No | Seconds between checks. Values at or below `0` default to `120`. |
| `min_impact` | No | Lowest incident impact to announce: `none`, `minor`, `major`, or `critical`. When omitted, all impacts are announced. |
| `services` | Yes | One or more service definitions to poll. |

The impact threshold applies globally to new incidents. Once an incident has
been announced, later updates and its resolution continue to be reported even
if its impact changes below the threshold.

### Service options

| Option | Required | Description |
| --- | --- | --- |
| `name` | Yes | Display name used in logs, Discord notifications, and persisted state. Keep it unique and stable. |
| `url` | Yes | Provider URL. The expected format depends on `provider`. |
| `provider` | Yes | Either `atlassian` or `statusio`. |
| `components` | No | Atlassian component names to monitor. When omitted, the entire service is monitored. |

Component names are matched case-insensitively and repeated whitespace is
ignored. A nested Atlassian component such as `API - Requests` can also match a
filter named `Requests`. Incidents with no component information are retained
so that potentially relevant service-wide events are not missed.

### Provider URLs

For an Atlassian-powered page, use its public base URL without a trailing slash:

```json
{
  "name": "OpenAI / ChatGPT",
  "url": "https://status.openai.com",
  "provider": "atlassian",
  "components": ["ChatGPT", "API"]
}
```

statuswatch appends `/api/v2/summary.json` automatically.

For Status.io, use the complete public status API endpoint:

```json
{
  "name": "Roblox",
  "url": "https://api.status.io/1.0/status/59db90dbcdeb2f04dadcf16d",
  "provider": "statusio"
}
```

Component filtering is currently supported only for Atlassian services.

## Discord webhook

In Discord, open **Server Settings → Integrations → Webhooks**, create a webhook
for the destination channel, and copy its URL into `webhook_url`. The webhook
URL grants permission to post to that channel, so do not commit or share your
real `config.json`.

Notifications include the service and incident name, current status, overall
service health, the latest provider update, and affected components when the
provider supplies them. Embed colors reflect the reported severity.

## State and notification behavior

statuswatch creates `statuswatch_state.json` beside the executable after the
first successful state change. It records announced incident IDs, their latest
update IDs, and services previously reported as unhealthy.

- An active incident is announced once when first observed.
- A new provider update triggers another notification.
- An incident disappearing from the active incident list is treated as resolved.
- An unhealthy overall status without an active incident triggers a degraded alert.
- A previously unhealthy service returning to normal triggers a recovery alert.
- Failed provider checks are logged and skipped until the next poll.
- State advances only after Discord accepts the corresponding webhook, allowing
  failed notifications to be retried later.

The first run can announce incidents that are already active. Deleting the state
file resets notification history and may cause active incidents or outages to be
announced again. The executable directory must be writable so state can be saved.

## Running continuously

You can keep statuswatch alive with any process manager. For example, with PM2:

```bash
pm2 start ./statuswatch --name statuswatch
pm2 save
```

Useful PM2 commands:

```bash
pm2 logs statuswatch
pm2 restart statuswatch
pm2 stop statuswatch
```

Make sure the compiled binary, `config.json`, and generated
`statuswatch_state.json` remain together when deploying or updating the app.

## Cross-compiling for Linux

From PowerShell on Windows:

```powershell
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o statuswatch ./src
Remove-Item Env:GOOS, Env:GOARCH
```

Copy `statuswatch` and `config.json` to the Linux host, then make the binary
executable:

```bash
chmod +x statuswatch
./statuswatch
```

Use `arm64` instead of `amd64` when targeting a 64-bit ARM server.

## Project structure

```text
statuswatch/
├── src/                    Go source code
├── config.example.json     Example service configuration
├── go.mod                  Go module definition
├── LICENSE                 MIT license
└── README.md               Project documentation
```

## License

This project is licensed under the **MIT License**. See [`LICENSE`](LICENSE) for
details.
