# Integration matrix

| App | Protocol | Status |
|---|---|---|
| GitLab, Harbor, Rancher, Grafana, Jenkins, Vault, NiFi, MinIO | OIDC | Point native OIDC config at IdpForge's discovery document (`/.well-known/openid-configuration`) |
| Windows / Active Directory | LDAP / Kerberos | Planned (LDAP sync) |
| macOS / Apple (Platform SSO) | OIDC/SAML via MDM | Planned |
| Microsoft 365 / Entra ID | SAML 2.0 | Planned (SAML IdP + SCIM) |
| Legacy apps with no SSO support | none | `/forwardauth` + Traefik `forwardAuth` middleware, injects `X-Forwarded-User` / `X-Forwarded-Groups` |

Registering a new OIDC application: `scripts/add-app.sh` / `scripts/add-app.ps1`.
