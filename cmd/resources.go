package main

//go:generate cmd-gen -type pod -import corev1 -hasNamespace=false
//go:generate cmd-gen -type namespace -import corev1 -hasNamespace=false
//go:generate cmd-gen -type clusterRole -import rbacv1 -hasNamespace=false
//go:generate cmd-gen -type clusterRoleBinding -import rbacv1 -hasNamespace=false
//go:generate cmd-gen -type node -import corev1 -hasNamespace=false
//go:generate cmd-gen -type persistentVolumeClaim -import corev1 -hasNamespace=false
//go:generate cmd-gen -type role -import rbacv1 -hasNamespace=false
//go:generate cmd-gen -type roleBinding -import rbacv1 -hasNamespace=false
//go:generate cmd-gen -type secret -import corev1 -hasNamespace=false
//go:generate cmd-gen -type service -import corev1 -hasNamespace=false
//go:generate cmd-gen -type serviceAccount -import corev1 -hasNamespace=false
//go:generate cmd-gen -type statefulSet -import appsv1 -hasNamespace=false
//go:generate cmd-gen -type deployment -import appsv1 -hasNamespace=false
