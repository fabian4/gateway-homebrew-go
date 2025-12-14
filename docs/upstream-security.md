# Upstream Security

The gateway supports configuring TLS settings per upstream service (cluster). This allows you to secure traffic between the gateway and your backend services.

## Configuration

You can configure TLS settings in the `services` section of your `config.yaml`.

### Modes

1.  **None (Plaintext)**: Default. Traffic is sent as HTTP/1.1 (or h2c) over TCP.
2.  **TLS (One-way)**: Gateway verifies the upstream's certificate.
3.  **mTLS (Mutual TLS)**: Gateway presents a client certificate to the upstream.

### Example Configuration

```yaml
services:
  # 1. Plaintext (default)
  - name: service-plaintext
    proto: http1
    endpoints:
      - "http://127.0.0.1:8080"

  # 2. TLS with public CA (e.g. upstream is https://example.com)
  - name: service-tls-public
    proto: http1
    endpoints:
      - "https://example.com"
    # No 'tls' block needed if using system root CAs and valid certs.
    # Just ensure endpoints start with https://

  # 3. TLS with self-signed cert or custom CA
  - name: service-tls-custom
    proto: http1
    endpoints:
      - "https://internal.local:8443"
    tls:
      # Option A: Skip verification (insecure)
      # insecure_skip_verify: true
      
      # Option B: Trust specific CA
      ca_file: "/etc/gateway/certs/internal-ca.crt"

  # 4. mTLS (Client Certificate)
  - name: service-mtls
    proto: http1
    endpoints:
      - "https://secure.local:8443"
    tls:
      ca_file: "/etc/gateway/certs/ca.crt"
      cert_file: "/etc/gateway/certs/client.crt"
      key_file: "/etc/gateway/certs/client.key"
```

## Reference

The `tls` block supports the following fields:

-   `insecure_skip_verify` (bool): If true, skips verification of the upstream certificate. **Not recommended for production.**
-   `ca_file` (string): Path to a PEM-encoded CA certificate file to trust.
-   `cert_file` (string): Path to a PEM-encoded client certificate file (for mTLS).
-   `key_file` (string): Path to a PEM-encoded client key file (for mTLS).
