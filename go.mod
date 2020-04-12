module github.com/thycotic/tss-k8s

require (
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a
	github.com/thycotic/tss-sdk-go v0.0.0-20200117214420-b62734ea7244
	k8s.io/api v0.0.0-20190720062849-3043179095b6
	k8s.io/apimachinery v0.0.0-20190719140911-bfcf53abc9f8
)

replace tss-k8s/pkg/server => ./pkg/server

go 1.13
