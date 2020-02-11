module github.com/gardener/nsxt-lb-provider

go 1.13

require (
	github.com/google/uuid v1.1.1
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/vmware/vsphere-automation-sdk-go/lib v0.1.1
	github.com/vmware/vsphere-automation-sdk-go/runtime v0.1.1
	github.com/vmware/vsphere-automation-sdk-go/services/nsxt v0.1.1
	golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	gopkg.in/gcfg.v1 v1.2.3
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v0.0.0
	k8s.io/cloud-provider v0.0.0
	k8s.io/component-base v0.0.0
	k8s.io/klog v0.4.0
	k8s.io/kubernetes v1.16.3
)

replace (
	gopkg.in/gcfg.v1 => github.com/mandelsoft/gcfg v1.2.4-0.20191118133708-71f203a139d3
	k8s.io/api => k8s.io/api v0.0.0-20191114100352-16d7abae0d2a // kubernetes-1.16.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191114105449-027877536833 // kubernetes-1.16.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb // kubernetes-1.16.3
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191114103151-9ca1dc586682 // kubernetes-1.16.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191114110141-0a35778df828 // kubernetes-1.16.3
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33 // kubernetes-1.16.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191114112024-4bbba8331835 // kubernetes-1.16.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191114111741-81bb9acf592d // kubernetes-1.16.3
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894 // kubernetes-1.16.3
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191114102325-35a9586014f7 // kubernetes-1.16.3
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac // kubernetes-1.16.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191114112310-0da609c4ca2d // kubernetes-1.16.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191114103820-f023614fb9ea // kubernetes-1.16.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191114111510-6d1ed697a64b // kubernetes-1.16.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191114110717-50a77e50d7d9 // kubernetes-1.16.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191114111229-2e90afcb56c7 // kubernetes-1.16.3
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191114113550-6123e1c827f7 // kubernetes-1.16.3
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191114110954-d67a8e7e2200 // kubernetes-1.16.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191114112655-db9be3e678bb // kubernetes-1.16.3
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191114105837-a4a2842dc51b // kubernetes-1.16.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191114104439-68caf20693ac // kubernetes-1.16.3
)
