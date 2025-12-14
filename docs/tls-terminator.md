# TLS Terminator (Inbound)

The gateway supports inbound TLS termination with SNI (Server Name Indication) support, allowing you to serve multiple domains with different certificates on the same port.

## Configuration

To enable TLS, add the `tls` section to your `config.yaml`.

```yaml
entrypoint:
  - name: web
    address: ":8443" # Standard HTTPS port

tls:
  enabled: true
  certificates:
    - cert_file: "/path/to/example.com.crt"
      key_file: "/path/to/example.com.key"
    - cert_file: "/path/to/another-domain.crt"
      key_file: "/path/to/another-domain.key"
```

## Features

- **SNI Support**: The gateway automatically selects the correct certificate based on the client's ClientHello SNI extension.
- **Multiple Certificates**: You can list multiple certificate pairs. The underlying Go `crypto/tls` library handles the selection logic.
- **TLS 1.2+**: The gateway enforces a minimum of TLS 1.2.

## Notes

- If `tls.enabled` is true, the server will expect HTTPS traffic on the configured listener address.
- Ensure the user running the gateway has read permissions for the certificate and key files.
