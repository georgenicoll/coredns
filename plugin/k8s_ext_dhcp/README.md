# k8s_ext_dhcp

## Name

*k8s_ext_dhcp* - resolves load balancer and external IPs from outside the Kubernetes clusters.

## Description

The functionality of this plugin is probably supported by the *k8s_external* plugin but I wanted to 
see what was needed to write a simple coredns plugin hence this!

This plugin allows resolution of an external address and zone into the external ip of a service (if it has one).
It can be used with something like metallb which assigns addresses from a pool of local IPs when a `LoadBalancer` type
service is set up.

This plugin also requires that the *kubernetes* plugin is also loaded as it relies on that plugin's API connection
to lookup the services.

The plugin currently only handles queries for A records, resolving names of the format `service-name.namespace.configured-zone`
to the external IP address on the service identified by `service-name.namespace` (if any).

## Example Configuration & Usage

Given the following configuration:

~~~
kubernetes:53 {
	health
	k8s_ext_dhcp {
		ttlSeconds          30
		continueOnNoMatch   false
	}
	kubernetes
}
~~~

and the following service in the `kube-system` namespace:

~~~
NAME             TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)         AGE
kube-dashboard   LoadBalancer   10.152.183.188   10.0.0.123    443:32557/TCP   24h
~~~

then the address `kube-dashboard.kube-system.kubernetes` will resolve to the IP `10.0.0.123`.

# Configuration

The `zone` will be taken from the enclosing zone in the Corefile

~~~
[zone][:port] {
	...
	k8s_ext_dhcp {
		ttlSeconds 			[ttl]
		continueOnNoMatch	[true|false]
	}
	kubernetes
	...
}
~~~

## `ttlSeconds`

- Number of seconds that the returned DNS record will live for
- Default: `120`

## `continueOnNoMatch`

- Set to `true` to continue trying to resolve the address in any following plugins that are configured.
- Set to `false` to return a resolution as soon as the service-name.namespace has not matched a service
with an external IP configured.
- Default: `true` 


## NOTE on local DNS configuration

The DNS server on the Router should be able to be set up to forward requests for the specific zone to be forwarded to coredns.

For example, openWrt allows the following configurations:

### `DNS forwardings`

To add an entry to forward requests in the `kubernetes` zone to the server at `10.0.0.123` Add an entry of the following format:
```
/kubernetes/10.0.0.123
```
### `Domain whitelist`

OpenWrt will drop any responses that contain an IP that is in the locally configured subnet with `possible DNS-rebind attack detected` unless a corresponding entry is added for the zone (in this example the entry `kubernetes` will need to be added).


# TODO

- Handle AAAA requests
- Add some unit tests
