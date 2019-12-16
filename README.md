# NSX-T Load Balancer Controller

*A Kubernetes load balancer provider for NSX-T. It can be used in parallel with
the regular `vSphere` cloud controller manager.*

This cloud controller manager implements (only) the load balancing API of
a cloud controller for an NSX-T environment. It uses by default a different
lease resource than the regular cloud controller manager.

There are two different modes:
- the *managed* mode manages the load balancer service. The NSX-T load balancer
  service is only created if it is required.
  This saves resources if no kubernetes service of type `LoadBalancer` is
  actually used.
- the *unmanaged* mode is used if the configuration specifies a load balancer service id. Here only the virtual servers are managed.
  
All generated NSX-T elements are tagged with the cluster name and the information
from the service. Using this tagging it is able to handle recovery
of lost or dangling elements and garbage collection of unused elements
previously generated by the controller, even if the kubernetes service object
is already (accidentally) gone.
  
### Load Balancer Classes

This load balancer controller supports the usage of multiple load balancer
classes. Every class may use another `IPPool` configured in NSX-T.
This supports the creation of load balancers in different visibility realms,
for example an `*internet facing* or a *private* load balancer. The
IPPools must be preconfigured in NSX-T. To select a dedicated loadbalancer
class the kunernetes service object must be annotated with the annotation:

```
loadbalancer.vsphere.class=<class name>
```

If no such annotation is given the default class will be used.

### Health Checks

For TCP load balancers a health check will be generated.

### Configuration File

The controller manager requires a configuration file:

```ini
[LoadBalancer]
ipPoolName = pool1
#ipPoolId = 0815
lbServiceId = 4711
size = MEDIUM

[LoadBalancerClass "public"]
ipPoolName = poolPublic

[LoadBalancerClass "private"]
ipPoolName = poolPrivate

[Tags]
tag1 = value1
tag2 = value2

[NSX-T]
user = admin
password = secret
host = nsxt-server
logicalRouterId = 1234
```

Only one of `ipPoolId` or `ipPoolName` may be given.
If the `lbServiceId` is given the controller is running in the *unmanaged*
mode. Otherwise the `logicalRouterId` must be given. If both
are given the configuration of the load balancer services is validated.

The `LoadBalancer` section defines an implicit default load balancer class. This
load balancer class is used if the service does not specify a dedicated
load balancer class via annotation. Its values are also used as defaults
for all explicitly specified load balancer classes.

Additionaly classes may be configured by the labeled `LoadBalancerClass`
sections.

The tags configured in the `Tags` section will be added as tags to all
generated elements in NSX-T.
