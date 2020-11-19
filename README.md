# trojan-go-exporter

trojan-go exporter

## Docker

```
docker run --rm -it zidy/trojan-go-exporter:0.0.1
```

## Trojan-go config

you need to make sure add api config to your config file. For example.

```json
{
  "api": {
    "enabled": true,
    "api_addr": "127.0.0.1",
    "api_port": 10000
  }
}
```
