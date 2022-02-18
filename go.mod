module github.com/thycotic/tss-k8s

require (
	github.com/mattbaird/jsonpatch v0.0.0-20200820163806-098863c1fc24
	github.com/thycotic/tss-sdk-go v1.1.0
	k8s.io/api v0.23.3
	k8s.io/apimachinery v0.23.3
)

replace tss-k8s/pkg/server => ./pkg/server

go 1.16
