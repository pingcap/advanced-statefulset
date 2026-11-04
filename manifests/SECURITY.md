# Controller permissions and deployment boundary

The controller's ClusterRole is restricted to its informer reads, Pod lifecycle,
PVC creation, ControllerRevision lifecycle, status updates and event writes.
`statefulsets/finalizers:update` is required by Kubernetes owner-reference
admission for `blockOwnerDeletion=true`; it must not be removed merely because
there is no explicit finalizers API call. Leader election uses a separate
namespace Role granting only Lease get/create/update. PV and Endpoints access,
CR creation/deletion, wildcard verbs/resources and pod subresources are absent.
The migration helper is a separate client operation; its permissions are not
part of the resident controller role.

The shipped controller watches all namespaces. Its ClusterRoleBinding reflects
that deployment model. Replacing it with namespace RoleBindings alone does not
make the current cluster-wide informer namespace-scoped. Keep the controller's
namespace, ServiceAccount and leader-election RoleBinding under platform admin
control; adjust the Role namespace if changing the leader-election namespace.

Treat the ability to create/update an Advanced StatefulSet (ASTS) as workload
administration in that namespace, like the builtin StatefulSet API. Its PodSpec
supports normal scheduling, volumes, service accounts and security contexts.
For deployments managed internally, these CRs are administered by trusted
operators; open-source users set policy in their own cluster. This controller
does not introduce a business-tenant privilege boundary or replace Kubernetes
Pod admission. Use Pod Security Admission and appropriate policy/RBAC to control
privileged Pods, host mounts and service-account selection. Do not grant ASTS
write to an untrusted actor while assuming its PodTemplate has fewer powers
than ordinary workload creation.

Similarly, ControllerRevision write access is workload-history administration.
Owner UID checks prevent accidental or stale-owner cross-workload handling;
they are not cryptographic provenance against a principal authorized to forge
or rewrite ControllerRevisions. Restrict such access to trusted administrators.
