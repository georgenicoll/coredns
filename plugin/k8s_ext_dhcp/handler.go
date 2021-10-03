/*
Package k8sExtDhcp implements external names for kubernetes clusters where external name is used to specify the
initial part of the dns name, for example an external name of dashboard in the zone kube.service will
return the external ip(s) when querying dashboard.kube.service.

This is probably functionality already supported by the k8s_external plugin but I wanted to try to write one... with problems.

Issues/Improvements:
- Make lookup constant time (index on external name?)
- Handle ipv6
- Testing is non-existant
- Others
*/
package k8sExtDhcp

import (
	"context"
	"net"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/kubernetes"
	"github.com/coredns/coredns/plugin/kubernetes/object"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

type Handler struct {
	next              plugin.Handler
	zones             []string
	ttlSeconds        uint32
	continueOnNoMatch bool
	kube              *kubernetes.Kubernetes
}

// New returns an initialized Handler.
func New(zones []string, ttlSeconds uint32, continueOnNoMatch bool) *Handler {
	h := new(Handler)
	h.zones = zones
	h.ttlSeconds = ttlSeconds
	h.continueOnNoMatch = continueOnNoMatch
	return h
}

// ServeDNS implements the plugin.Handler interface.
func (h Handler) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	if h.kube == nil {
		return plugin.NextOrFailure(h.Name(), h.next, ctx, w, r)
	}

	state := request.Request{W: w, Req: r}

	qname := state.QName()
	zone := plugin.Zones(h.zones).Matches(qname)

	if zone == "" {
		return plugin.NextOrFailure(h.Name(), h.next, ctx, w, r)
	}

	//find the service part (this is the part prior to the zone, minus a separating '.' [hence the -1])
	serviceName := qname[:len(qname)-len(zone)-1]

	log.Debugf("k8s ext: qname=%s, zone=%s, service=%s", qname, zone, serviceName)

	//Find the first service with a label that matches...
	//NOTE:  should make this more efficient in a real implementation
	//NOTE:  this noddy implementation only handles ipv4
	for _, svc := range h.kube.APIConn.ServiceList() {
		if serviceName == svc.ExternalName && len(svc.ExternalIPs) > 0 {
			records := h.serviceRecords(svc, &w, &state)
			extra := make([]dns.RR, 0)

			if len(records) == 0 {
				return plugin.BackendError(ctx, h.kube, zone, dns.RcodeSuccess, state, nil, plugin.Options{})
			}

			m := new(dns.Msg)
			m.SetReply(r)
			m.Authoritative = true
			m.Answer = append(m.Answer, records...)
			m.Extra = append(m.Extra, extra...)
			w.WriteMsg(m)
			return dns.RcodeSuccess, nil
		}
	}

	//Nothing found call the next or fail depending on settings
	if h.continueOnNoMatch {
		return plugin.NextOrFailure(h.Name(), h.next, ctx, w, r)
	}
	return plugin.BackendError(ctx, h.kube, zone, dns.RcodeSuccess, state, nil, plugin.Options{})
}

// Name implements the Handler interface.
func (h Handler) Name() string { return pluginName }

//Populates a record for each external ip in the service
func (h Handler) serviceRecords(service *object.Service, w *dns.ResponseWriter, request *request.Request) (records []dns.RR) {

	for _, ipAddress := range service.ExternalIPs {
		switch request.QType() {
		case dns.TypeA:
			ip := net.ParseIP(ipAddress)
			a := &dns.A{Hdr: dns.RR_Header{Name: request.QName(), Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: h.ttlSeconds}, A: ip}
			records = append(records, a)
		case dns.TypeAAAA:
			// FIXME:  add to handle ipv6
		case dns.TypeCNAME:
			continue
		}

	}

	return records
}
