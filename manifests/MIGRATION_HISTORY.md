# ControllerRevision migration identities

The builtin StatefulSet and the new Advanced StatefulSet have different UIDs.
The upgrade helper records the source builtin UID in
`apps.pingcap.com/upgrade-from-statefulset-uid` on the target and marked revisions.
The controller only admits still-builtin-owned migration history when that
source UID, owner name/kind/API version and migration marker agree. The upgrade
helper marks only revisions actually controlled by the source UID, including
previously marked revisions left by an interrupted older helper. Source deletion
uses a UID precondition, so a same-name replacement cannot be deleted.

Normal selector-matched orphan adoption remains compatible with Kubernetes
StatefulSet behavior. The additional marker-only list requires the source
binding for orphans as well; an arbitrary upgrade label is insufficient. Empty
selectors are rejected rather than listing/adopting all namespace history.

For migrations already started by an old helper without UID metadata, a
still-builtin-owned revision named by current/update status causes reconciliation
to wait for garbage collection to orphan it. The controller then preserves that
explicitly referenced orphan, adopts it and restores selector labels. It does
not treat a missing current history as permission to silently switch to the
update revision. Unreferenced legacy marker-only orphans are not automatically
trusted; administrators may need to review old rollback history separately.
If retrying an interrupted helper before source deletion, use the upgraded
helper so it binds all source-owned marked history before GC.

Use matching controller/client upgrades to gain both sides of the migration
binding. No existing objects are rewritten at installation. Restrict CR,
ControllerRevision and migration-annotation write access to trusted workload
administrators; UID metadata protects ownership consistency, not against a
principal already authorized to rewrite history/provenance fields.
