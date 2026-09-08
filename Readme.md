# TLS-Client-API

### Preface

This is an application which is using [gosoline](https://github.com/justtrackio/gosoline) and [TLS-Client](https://github.com/bogdanfinn/tls-client) to run a simple request forwarding service with the option to use specific tls fingerprints which are implemented in [TLS-client](https://github.com/bogdanfinn/tls-client).

### WebSocket Bridge (`GET /api/ws`)

`/api/ws` is a generic WebSocket-to-WebSocket bridge:

```
client WebSocket -> local WebSocket -> tls-client-api -> tls-client WebSocket -> remote WSS target
```

It exists so a remote WebSocket connection can use tls-client's TLS fingerprinting, proxy
support and client profiles. The bridge only transports WebSocket application messages
(text/binary) byte-for-byte;

#### Authentication

The endpoint is protected by the same `x-api-key` authentication as every other endpoint. The
key is a local bridge concern only and is never forwarded to the remote target.

#### Requesting a connection

The Node client performs a normal WebSocket Upgrade request to `/api/ws` with two headers:

- `x-api-key`: the existing local bridge API key.
- `x-tls-client-ws-config`: `Base64URL(JSON.stringify(config))` (no padding).

Config schema (version 1):

```jsonc
{
  "version": 1,
  "requestUrl": "wss://example.com/socket.io/?EIO=4&transport=websocket",
  "tlsClientIdentifier": "chrome_136",
  "proxyUrl": "http://127.0.0.1:40001",
  "headers": {
    "Origin": "https://example.com",
    "Cookie": ["a=1", "b=2"]
  },
  "headerOrder": [
    "host",
    "upgrade",
    "connection",
    "origin"
  ],
  "protocols": [],
  "handshakeTimeoutMilliseconds": 10000,
  "readBufferSize": 0,
  "writeBufferSize": 0,
  "insecureSkipVerify": false,
  "withRandomTLSExtensionOrder": false,
  "disableIPV4": false,
  "disableIPV6": false,
  "serverNameOverwrite": ""
}
```

- `version`, `requestUrl` and `tlsClientIdentifier` are required. `requestUrl` must use `ws://`
  or `wss://`.
- `headers` values may be a single string or an array of strings; both forms are normalized
  into the remote request headers (these are headers sent to the **remote** target, e.g.
  `Origin`, `User-Agent`, `Cookie` - never `x-api-key` or `x-tls-client-ws-config`).
- `headerOrder` controls the header write order sent to the remote target.
- `protocols` are the WebSocket subprotocols requested from the remote target. If the remote
  selects one, the same protocol is echoed back on the local Upgrade response.
- The remote HTTP client always forces HTTP/1.1, as required by tls-client's WebSocket support.
- On a successful upgrade, the local response also carries an informational
  `x-tls-client-ws-protocol: 1` header. It is metadata only - clients must not depend on it.

##### Deprecated field aliases

For backward compatibility, the following pre-canonicalization field names are still accepted.
They only take effect when the canonical field is **absent** from the payload; if both are
present, the canonical field always wins. New integrations should only use the canonical names.

| Deprecated alias (do not use) | Canonical field |
|---|---|
| `withRandomTlsExtensionOrder` | `withRandomTLSExtensionOrder` |
| `disableIPv4` | `disableIPV4` |
| `disableIPv6` | `disableIPV6` |


#### Connection lifecycle

The local connection is **not** upgraded immediately. tls-client-api first connects to the
remote target using tls-client (TLS fingerprint, proxy, etc.), and only upgrades the local
Node connection to HTTP 101 once the remote WebSocket handshake has succeeded. This means the
Node WebSocket never reports `open` before the remote target is actually connected. If the
remote handshake fails, the request is rejected with a plain HTTP error response (400/401/403,
502 or 504) and the local connection is never upgraded.

After both sides are connected, messages are relayed unmodified and concurrently in both
directions, preserving message type (text/binary) and close codes/reasons. No bridge status
JSON is ever injected into the data stream.

### Detailed Documentation

https://bogdanfinn.gitbook.io/open-source-oasis/standalone-api-application

### Questions?

Join my discord support server for free: https://discord.gg/7Ej9eJvHqk
No Support in DMs!


### Appreciate my work?

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/CaptainBarnius)

