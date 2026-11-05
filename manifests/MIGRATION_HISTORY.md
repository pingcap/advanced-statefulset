# ControllerRevision ownership and migration compatibility

The controller accepts history only when it has no controller owner or its
controller owner UID matches the target Advanced StatefulSet (ASTS). A revision
with another controller owner is never used, adopted, updated or deleted by a
normal ASTS reconcile. Empty selectors are rejected before listing history.

## Migration without an Operator/controller version pairing

The existing `apps.pingcap.com/upgrade-to-asts` label remains the migration
protocol. Old helper versions and manual migrations can keep using it:

1. Mark source history and remove its selector labels as the old helper does.
2. Create the ASTS and copy current/update status when available.
3. Delete the builtin StatefulSet with orphan propagation.
4. The ASTS controller waits while marked history still has its builtin owner.
   Once GC removes that owner, it adopts all marked orphan history, including
   older revisions not referenced by current/update status, restores selector
   labels and clears the migration marker. Normal `revisionHistoryLimit` retention
   applies after adoption.

Waiting is necessary even when source UID metadata is present: metadata does
not authorize using another controller's data. A referenced foreign revision
also blocks reconciliation instead of silently replacing current history with
a new template. GC must finish orphaning before migration reconciliation can
continue. Do not remove the legacy migration label before adoption finishes.

`apps.pingcap.com/upgrade-from-statefulset-uid` is optional bookkeeping for the
new helper's completed-migration retry check. Controller history admission does
not depend on it. An Operator may replace ASTS annotations without breaking
history adoption; old helpers do not need to produce the annotation.

The new helper independently checks the live source UID, marks only history
actually owned by that source (including previously marked history on retry),
and uses a UID precondition when deleting the source. These helper safeguards
are obtained by upgrading that client, but are not prerequisites for running
this controller with an older Operator. Already adopted history needs no rewrite.

## Required authorization boundary

Orphan adoption preserves the existing Kubernetes-style behavior. Labels,
ownerReferences, status revision names and optional UID annotations are not
proof against a principal authorized to forge them. This change prevents
cross-owner handling; it does not provide hostile-workload isolation inside a
shared namespace by itself.

Treat ControllerRevision writes and ASTS/status writes as trusted workload
administration. Allow the actual built-in workload controllers, ASTS controller,
Operator/migration identity and designated administrators to perform their
required operations. Untrusted tenants must not have arbitrary create/update/
patch/delete on ControllerRevisions or update/patch on ASTS/status; ASTS writes
must likewise follow the workload-administration trust model.

Audit all effective RoleBindings/ClusterRoleBindings, including inherited or
wildcard grants. RBAC is additive: narrowing the ASTS controller's own role does
not revoke another principal's permissions. When resource-level RBAC cannot
express the desired writer/object restrictions, enforce them with an admission
policy or webhook using authenticated request identity. Protect that policy and
the controller/Operator ServiceAccounts from tenant modification. Users with
namespace-admin or equivalent workload administration rights are trusted in this
model. Stronger isolation requires a separate enforced authorization boundary.

No live permissions are changed by this repository documentation. Operators and
controller versions need not be paired, but the deployment must enforce the
trusted-writer boundary before treating legacy orphan migration as safe against
untrusted writers. Existing history created before that boundary was enforced
may require administrator review.
