# Security policy

## Supported versions

Security fixes are applied to the latest release and the default branch.

## Reporting a vulnerability

Please do not open a public issue for a suspected vulnerability. Use GitHub's
private vulnerability reporting feature on the repository's **Security** tab.
Include the affected version, reproduction steps, potential impact, and any
suggested mitigation.

If private reporting is not enabled, ask a maintainer for a private contact
channel without including sensitive details in the public request.

You should receive an acknowledgement within seven days. Please allow time for
the report to be validated and a fix to be prepared before public disclosure.

## Security boundaries

The application makes a read-only HTTP request to the configured RSS endpoint
and prints selected feed data. It does not store credentials or financial data.
Treat custom feed URLs as untrusted: only use endpoints you expect to contact.
