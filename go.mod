module github.com/gardener/nsxt-lb-provider

go 1.13

require (
	github.com/pkg/errors v0.8.0
	github.com/spf13/cobra v0.0.0-20180319062004-c439c4fa0937
	github.com/spf13/pflag v1.0.3
	github.com/vmware/go-vmware-nsxt v0.0.0-20190201205556-16aa0443042d // master
	gopkg.in/gcfg.v1 v1.2.3
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v0.0.0
	k8s.io/cloud-provider v0.0.0
	k8s.io/component-base v0.0.0
	k8s.io/klog v0.4.0
	k8s.io/kubernetes v1.15.4
)

replace (
	github.com/vmware/go-vmware-nsxt => ../go-vmware-nsxt
	gopkg.in/gcfg.v1 => github.com/mandelsoft/gcfg v1.2.4-0.20191118112722-f0b8a49fc708
	k8s.io/api => k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2 // kubernetes-1.15.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190918201827-3de75813f604 // kubernetes-1.15.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d // kubernetes-1.15.4
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190918200908-1e17798da8c1 // kubernetes-1.15.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190918202139-0b14c719ca62 // kubernetes-1.15.4
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a // kubernetes-1.15.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190918203125-ae665f80358a // kubernetes-1.15.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190918202959-c340507a5d48 // kubernetes-1.15.4
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b // kubernetes-1.15.4
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190918200425-ed2f0867c778 // kubernetes-1.15.4
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190817025403-3ae76f584e79 // kubernetes-1.15.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20190918203248-97c07dcbb623 // kubernetes-1.15.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190918201136-c3a845f1fbb2 // kubernetes-1.15.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190918202837-c54ce30c680e // kubernetes-1.15.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190918202429-08c8357f8e2d // kubernetes-1.15.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190918202713-c34a54b3ec8e // kubernetes-1.15.4
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190918202550-958285cf3eef // kubernetes-1.15.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190918203421-225f0541b3ea // kubernetes-1.15.4
	k8s.io/metrics => k8s.io/metrics v0.0.0-20190918202012-3c1ca76f5bda // kubernetes-1.15.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190918201353-5cc279503896 // kubernetes-1.15.4
)
