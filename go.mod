module github.com/giantswarm/aws-collector

go 1.14

require (
	github.com/aws/aws-sdk-go v1.38.2
	github.com/giantswarm/apiextensions/v2 v2.6.2
	github.com/giantswarm/exporterkit v0.2.1
	github.com/giantswarm/k8sclient/v4 v4.1.0
	github.com/giantswarm/microendpoint v0.2.0
	github.com/giantswarm/microerror v0.3.0
	github.com/giantswarm/microkit v0.2.2
	github.com/giantswarm/micrologger v0.5.0
	github.com/giantswarm/operatorkit/v2 v2.0.2
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus/client_golang v1.10.0
	github.com/spf13/afero v1.3.1 // indirect
	github.com/spf13/viper v1.7.1
	go.uber.org/zap v1.15.0 // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v0.18.9
	sigs.k8s.io/cluster-api v0.3.8
)

replace (
	github.com/coreos/etcd v3.3.10+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/coreos/etcd v3.3.13+incompatible => github.com/coreos/etcd v3.3.25+incompatible
)
