# Jenkins pipeline status

The repository maintainers confirmed on 2026-11-04 that the legacy Jenkins PR
verification pipeline is no longer used. Its `bootstrap.groovy` and
`build.groovy` have been removed rather than kept as supported deployment
examples. They loaded PR-controlled Groovy in the Jenkins job context and ran
PR code in privileged Kubernetes agents with host mounts.

PR verification and kind E2E use the existing `.github/workflows/ci.yaml`
GitHub Actions workflow. The removed scripts remain in Git history for historical
investigation, not as supported deployment examples. This repository change does
not modify external Jenkins job configuration or revoke credentials.

`release.groovy` is a separate release pipeline. This retirement confirmation
covers only the legacy PR tests; it makes no claim that release jobs are unused
or that their credentials have been revoked. The release pipeline is unchanged.
