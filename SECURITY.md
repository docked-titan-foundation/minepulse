# Security Policy

## Reporting

Do not open public issues for security problems. Email
<security.emporium706@passmail.com>. We aim to acknowledge within 48 hours and follow up
within 7 days.

## Supported Versions

| Version | Supported |
| ------- | --------- |
| 1.x     | ✅        |
| 0.x     | ❌        |

## Scope

This policy covers the `minepulse` tool itself. Because minepulse is designed to be
strictly read-only toward a cluster (Constitution II), the highest-severity classes we
care about are:

- any path by which minepulse could **mutate** cluster or miner state (it must not);
- privilege beyond the documented read verbs in `deploy/rbac.yaml`;
- exfiltration of data to anywhere other than the cluster API server and the configured
  pool API (Constitution VII) — there is no telemetry;
- supply-chain integrity of released binaries/images (they are cosign-signed and
  SBOM-attested; report anything that undermines verification).

The wallet address minepulse displays is public information by design. Vulnerabilities
in upstream dependencies (client-go, Bubble Tea, XMRig, the pool) should also be reported
to their maintainers.
