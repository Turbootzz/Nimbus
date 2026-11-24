# Nimbus

Nimbus is a self-hosted dashboard for the apps and services in your homelab.

- **Health checks**: see what is online, with uptime history and response times
- **Widgets** next to your services: clock, weather, notes, bookmarks, RSS, calendar and more
- **Integrations**: live numbers from Sonarr, Radarr, qBittorrent, Pi-hole, AdGuard Home, Proxmox, Jellyfin, Home Assistant and Uptime Kuma on their tile
- **Multi-user**: every user has their own dashboard, wallpaper and theme
- **Sign-in** with a local account, Google, GitHub, Discord or your own OpenID Connect provider

## First start

Open Nimbus and create your account. The first account becomes the admin.

Nimbus generates its secrets on first start and keeps them in the app data folder (`data/uploads/.secrets`). Include that folder in your backups: without the encryption key, saved integration credentials can't be read.

## Links

- [Source and documentation](https://github.com/Turbootzz/Nimbus)
- [Configuration](https://github.com/Turbootzz/Nimbus/blob/main/docs/CONFIGURATION.md)
