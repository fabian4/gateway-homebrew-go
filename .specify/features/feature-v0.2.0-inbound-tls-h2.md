# Feature: v0.2.0 - Inbound TLS & HTTP/2/gRPC

Goals
- Terminate TLS for multiple hosts; enable ALPN for h2/http1.1; support gRPC pass-through.

Scope
- TLS termination (SNI, multiple certs)
- ALPN: h2/http1.1 negotiation
- Basic gRPC pass-through (no transcoding)

Requirements
- Cert Management: PEM files path per host; hot reload optional (defer if complex).
- Cipher Suites: sane defaults; configurable list; disable insecure ciphers.
- ALPN: server offers [h2, http/1.1]; protocol-specific handlers.
- gRPC: pass-through over h2, preserve metadata; limit max message size and enforce deadlines.

Config Additions
- tls: { hosts: [{ host, certFile, keyFile }], minVersion: TLS1.2, cipherSuites?: [..] }
- alpn: { enabled: true, protocols: ["h2","http/1.1"] }
- grpc: { passthrough: true, maxMsgBytes?: 4_194_304, enforceDeadline?: true }

Acceptance Criteria
- Gateway terminates TLS using SNI and serves distinct certs per host.
- ALPN negotiates h2 vs http/1.1 as appropriate.
- gRPC requests are proxied end-to-end without modification.

Testing
- TLS: cert selection by SNI, handshake success.
- ALPN: clients negotiate h2/http1.1; protocol-specific behavior validated.
- gRPC: sample service roundtrip over h2; headers and status preserved.

Operational & NFR
- Security: no RSA < 2048; prefer ECDSA; disable TLSv1.0/1.1.
- Performance: h2 multiplexing validated with concurrent streams.
- Observability: TLS handshake failures counter; gRPC deadline violations counter.

Risks & Rollout
- Risk: misconfigured certs break hosts; staged rollout per host.
- Rollout: enable TLS for single host, verify, then expand; turn on h2 after baseline.
