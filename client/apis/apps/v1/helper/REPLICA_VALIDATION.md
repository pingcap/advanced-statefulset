# Checked replica helpers

The controller rejects malformed/null delete-slots annotations, negative slots,
nil/negative/oversized replicas, and an effective ordinal range over 10,000,
before adoption or reconciliation writes. Both CRD versions limit replicas to
10,000/int32; controller validation also protects objects already stored and
updates through scale. Ignored out-of-range positive slots retain their existing
meaning; duplicates are deduplicated. The input array is also limited to 10,000
entries. Invalid configuration is an error, not an empty delete-slots set.

The ordinal-range budget includes effective deleted slots, not just the desired
Pod count. For example, replicas=10,000 with delete-slots=[0] is rejected because
it would materialize a range of 10,001. Before upgrading, correct invalid objects
or reduce/split workloads exceeding this limit. Rejected existing workloads keep
their Pods; reconciliation resumes after administrators correct the CR. The
controller does not silently clamp replicas or interpret invalid data as scale-in.

New callers reading user-controlled configuration should use
`GetDeleteSlotsChecked`, `GetMaxReplicaCountAndDeleteSlotsChecked` and
`GetPodOrdinalsChecked`, propagate errors and avoid performing writes on error.
`AddDeleteSlots` now refuses to overwrite a malformed existing annotation.

Legacy no-error helper signatures are retained for source compatibility and
must only be used with already validated input. Their old calculation behavior
is not a safe validation boundary; downstream operators need to adopt checked
APIs to obtain equivalent protection. This change fixes the resident controller
and offers safe library entrypoints; it does not silently change an external
caller's malformed input into an empty desired Pod set.
