package main

//go:generate cmd-gen -type pod -hasNamespace=false
//go:generate cmd-gen -type namespace -hasNamespace=false
//go:generate cmd-gen -type clusterRole -hasNamespace=false
//go:generate cmd-gen -type clusterRoleBinding -hasNamespace=false
//go:generate cmd-gen -type node -hasNamespace=false
//go:generate cmd-gen -type persistentVolumeClaim -hasNamespace=false
//go:generate cmd-gen -type role -hasNamespace=false
//go:generate cmd-gen -type roleBinding -hasNamespace=false
//go:generate cmd-gen -type secret -hasNamespace=false
//go:generate cmd-gen -type service -hasNamespace=false
//go:generate cmd-gen -type serviceAccount -hasNamespace=false
//go:generate cmd-gen -type statefulSet -hasNamespace=false

var number int
