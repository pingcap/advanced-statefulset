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

On 2026-11-09, the repository maintainer approved retiring the legacy Jenkins
release integration. The GitHub Actions `release` workflow was already in the
`disabled_manually` state. Its workflow file (`.github/workflows/release.yaml`),
remote trigger helper (`hack/trigger-jenkins-job.py`), and Jenkins pipeline
(`hack/jenkins/release.groovy`) have been removed. They remain in Git history for
historical investigation, not as supported release tooling.

This removal does not disable the external `release_advanced_statefulset` Jenkins
job or revoke its credentials. The job's status and last build could not be
verified because its configured host, `internal.pingcap.net`, did not resolve.
The Jenkins administrator still needs to confirm the job is disabled and its
remote trigger token is revoked, and remove any job-specific credential access.
The GitHub Actions secret `JENKINS_TRIGGER_TOKEN` is no longer used by this
repository and can be removed. Shared credentials must be reviewed for other
consumers before revocation.
