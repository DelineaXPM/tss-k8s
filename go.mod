module github.com/DelineaXPM/tss-k8s/v2

require (
	github.com/DelineaXPM/tss-sdk-go/v2 v2.0.0
	github.com/mattbaird/jsonpatch v0.0.0-20200820163806-098863c1fc24
	k8s.io/api v0.26.0
	k8s.io/apimachinery v0.26.0
)

replace tss-k8s/pkg/server => ./pkg/server

go 1.16
