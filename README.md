# WEDOS IP ranges module for Caddy

This project is a fork of `caddy-cloudflare-ip` (upstream: https://github.com/WeidiDeng/caddy-cloudflare-ip) adapted to fetch **WEDOS Global / WEDOS Protection** origin-facing IP ranges for use with Caddy's `trusted_proxies`.

The module downloads the current list from:

- `https://ips.wedos.global/ips.txt`

It implements Caddy's `http.ip_sources` interface.

## Caddy module ID

- `http.ip_sources.wedos`

## Example config

Put the following config in your **global options** under the corresponding server options:

```caddyfile
{
  servers {
    trusted_proxies combine {
      wedos {
        interval 12h
        timeout 15s
      }
    }

    trusted_proxies_strict
    client_ip_headers X-Forwarded-For X-Real-IP
  }
}
```

## Defaults

| Name     | Description                                    | Type     | Default    |
|----------|------------------------------------------------|----------|------------|
| interval | How often the WEDOS IP list is refreshed       | duration | 1h         |
| timeout  | Maximum time to wait for a response from WEDOS | duration | no timeout |

## Notes

- WEDOS may change IP ranges over time; this module refreshes them periodically.
- `ips.txt` may be whitespace-separated; the module parses it as tokens.

## License

Apache License 2.0 (same as the upstream project).
