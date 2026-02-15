# caddy-mss

Add expose TCP MSS in `X-Client-MSS` header.

## Caddyfile

```caddyfile
{
    # Register the handler order
   	order mss_header first
}

:8080 {
    # enable middleware
    mss_header
}
```
