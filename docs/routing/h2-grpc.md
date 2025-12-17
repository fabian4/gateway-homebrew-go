# HTTP/2 & gRPC

## ALPN

The gateway automatically negotiates the application-layer protocol using ALPN (Application-Layer Protocol Negotiation) during the TLS handshake.

- **Supported Protocols**: `h2` (HTTP/2) and `http/1.1`.
- **Configuration**: No additional configuration is required. If TLS is enabled, ALPN is active.
- **Behavior**:
    - If the client supports `h2`, the connection is upgraded to HTTP/2.
    - Otherwise, it falls back to `http/1.1`.

## gRPC Pass-through

The gateway supports basic gRPC pass-through by:
- Supporting HTTP/2 (via ALPN).
- Preserving `TE: trailers` header.
- Flushing response headers immediately.
- Copying response trailers to the downstream client.

This allows standard gRPC unary and streaming calls to work transparently.
