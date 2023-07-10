module github.com/DelineaXPM/tss-k8s/v2

require (
	github.com/DelineaXPM/tss-sdk-go/v2 v2.0.0
	github.com/mattbaird/jsonpatch v0.0.0-20230413205102-771768614e91
	k8s.io/api v0.27.3
	k8s.io/apimachinery v0.27.3
)

replace tss-k8s/pkg/server => ./pkg/server

go 1.16
