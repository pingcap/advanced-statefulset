# PR pipeline trust boundary

The Jenkins job must obtain `bootstrap.groovy` from the trusted master branch or
an administrator-pinned trusted commit. Never configure the job's Script Path to
load the bootstrap itself from the PR head. Repository changes cannot enforce
that external Jenkins setting. Update the job to the repaired bootstrap before
considering the deployed pipeline remediated.

The bootstrap checks out only master into a clean, separate directory and loads
`build.groovy` there. A PR supplies only a validated full commit SHA. Public HTTPS
checkouts use no deploy/registry credentials. Verify, unit/build and integration
commands still execute PR code, in separate non-root Kubernetes agents with no
hostPath, Docker socket, privileged capability or service-account token. Do not
attach deployment credentials, shared persistent workspaces or trusted-job cloud
identity to these PR agents. Keep the Jenkins agent-to-controller security
boundary enabled and keep trusted/release agents separate.

The former kind/Docker-in-Docker E2E stages are deliberately removed from the
Jenkins PR pipeline: privileged nested Docker plus host mounts is not a safe
sandbox for arbitrary PR code. The repository's existing GitHub Actions workflow
runs kind E2E on disposable hosted VMs. Preserve those required checks. If Jenkins
E2E is required, provision a separately isolated disposable VM runner rather than
restoring privileged Pods or host mounts. Historical v1.12/v1.16 Jenkins-only
matrix coverage is not retained by this change.

The agent's Go version matches the repository's current go.mod. The release
pipeline is separate and is not invoked or deployed by this PR change. Tests use
Groovy pipeline stubs; they do not substitute for validating Kubernetes/Jenkins
plugin behavior in the deployed instance.
