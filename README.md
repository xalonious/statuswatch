# statuswatch
A lightweight self-hosted tool that monitors third-party service status pages and sends Discord notifications when incidents are detected, updated, or resolved.  
Built to run continuously on a server or VPS via pm2, with zero dependencies beyond the compiled binary.

## Overview
statuswatch polls the official status pages of configured services and fires Discord webhook alerts when something changes — new incident, update, resolution, degraded performance, or recovery. State is persisted between runs so you never get duplicate alerts.

## Features
- Monitors **Atlassian Statuspage** and **Status.io** based services
- Optional per-service **component filtering** — only get alerted about what you care about
- Optional **minimum impact threshold** — filter out minor incidents and only get alerted for what matters
- Discord embeds for new incidents, updates, resolutions, degraded services, and recoveries
- Single self-contained binary, no runtime or dependencies required on the server
- Cross-compiles for Linux from any machine

For configuration, see `config.example.json`.

## License
This project is licensed under the **MIT License**.