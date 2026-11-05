# Checked replica helpers

The controller rejects malformed delete-slots annotations, negative slots,
nil/negative replicas and effective ordinal arithmetic that overflows int32.
Validation runs before revision adoption, Pod claiming or reconciliation writes.
Both CRD versions constrain replicas to non-negative int32 values. There is no
fixed replica, ordinal-range or annotation-item limit of 10,000.

The existing non-negative slot semantics are preserved: duplicate slots are
deduplicated, positive slots outside the effective range are ignored, JSON
`null` means no slots, and null array elements mean ordinal zero. The null
semantics match Go's `[]int32` decoding used by TiDB Operator v1. For example,
`replicas=3` with `[null,2]` has desired ordinals `[1,3,4]`.

TiDB Operator v1 can request extra replicas for failover or temporary replicas
for volume replacement. These continue to reconcile above 10,000 replicas, as
do workloads whose deleted slots move the effective ordinal range above 10,000.
Negative slots and int32 arithmetic overflow are unsafe and remain errors;
they are never clamped or interpreted as an empty desired set. Existing objects
with those errors keep their Pods until the configuration is corrected.

The controller indexes existing Pods and tracks at most one missing ordinal
in OrderedReady mode or 500 missing ordinals in Parallel mode per reconciliation.
Pod templates are copied individually immediately before creation; the batch
retains only identity/revision placeholders for missing Pods.
It limits Pod control operations to 500 per pass, records partial status and
requeues unfinished batches. This is a work budget, not a configuration limit.
Memory and traversal depend on existing Pods, supplied slots and the batch,
not on a potentially enormous declared replica range. OrderedReady health
checks, scale-down order and rolling-update partitions keep their semantics.

New callers reading user-controlled configuration should use
`GetDeleteSlotsChecked` and `GetMaxReplicaCountAndDeleteSlotsChecked`, propagate
errors and avoid writes on error. For incremental ordinal traversal, use
`VisitPodOrdinalsChecked` and return false from the visitor to stop early. It
validates the entire configuration before the first callback and does not
materialize the desired set. `AddDeleteSlots` refuses to overwrite malformed
or negative existing annotations.

Legacy no-error helper signatures retain source compatibility and calculation
behavior. They are not safe validation boundaries; their set-returning ordinal
helpers still allocate the entire result. Downstream operators, including the
copies of these algorithms in TiDB Operator v1, need separate migration to
checked validation and incremental/bounded work. Updating this controller or
only bumping the client dependency does not harden those downstream paths.
