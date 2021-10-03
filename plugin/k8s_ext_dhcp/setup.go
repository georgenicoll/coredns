package k8sExtDhcp

import (
	"strconv"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/kubernetes"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

const pluginName = "k8s_ext_dhcp"

var log = clog.NewWithPlugin(pluginName)

func init() { plugin.Register(pluginName, setup) }

func setup(c *caddy.Controller) error {
	log.Debugf("Setting up %s", pluginName)

	//TODO: Load Configuration
	h, err := parse(c)
	if err != nil {
		return plugin.Error(pluginName, err)
	}

	// Do this in OnStartup, so all plugins have been initialized.
	c.OnStartup(func() error {
		m := dnsserver.GetConfig(c).Handler("kubernetes")
		if m == nil {
			return nil
		}
		h.kube = m.(*kubernetes.Kubernetes)
		return nil
	})

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		h.next = next
		return h
	})

	return nil
}

func parse(c *caddy.Controller) (*Handler, error) {
	zones := make([]string, 0)
	var ttlSeconds uint32 = 120
	continueOnNoMatch := true

	for c.Next() { //k8s_ext_dhcp
		zones = plugin.OriginsFromArgsOrServerBlock(c.RemainingArgs(), c.ServerBlockKeys)
		for c.NextBlock() {
			switch c.Val() {
			case "ttlSeconds":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, c.ArgErr()
				}
				_64bitTtl, err := strconv.Atoi(args[0])
				if err != nil {
					return nil, c.ArgErr()
				}
				ttlSeconds = uint32(_64bitTtl)
			case "continueOnNoMatch":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, c.ArgErr()
				}
				conm, err := strconv.ParseBool(args[0])
				if err != nil {
					return nil, c.ArgErr()
				}
				continueOnNoMatch = conm
			default:
				return nil, c.ArgErr()
			}
		}
	}

	log.Debugf("Setting up handler: zones=%s, ttlSeconds=%s, continueOnNoMatch=%s", zones, ttlSeconds, continueOnNoMatch)

	e := New(zones, ttlSeconds, continueOnNoMatch)
	return e, nil
}
