package kubernetes

// TODO (rk) refactor existing resource creation to these generated files
//  NB. the current CreateXXX calls are not used, they used the old pkg/client implementation
//  which needs to be removed as part of this rework

//go:generate stub-gen -api RbacV1 -type ClusterRole -hasNamespace=false
//go:generate stub-gen -api RbacV1 -type ClusterRoleBinding -hasNamespace=false
//go:generate stub-gen -api CoreV1 -type Node -hasNamespace=false
//go:generate stub-gen -api CoreV1 -type PersistentVolumeClaim
//go:generate stub-gen -api CoreV1 -type Pod
//go:generate stub-gen -api RbacV1 -type Role
//go:generate stub-gen -api RbacV1 -type RoleBinding
//go:generate stub-gen -api CoreV1 -type Secret
//go:generate stub-gen -api CoreV1 -type Service
//go:generate stub-gen -api CoreV1 -type ServiceAccount
//go:generate stub-gen -api AppsV1 -type StatefulSet
