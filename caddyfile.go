// Modified by hexband in 2025: adapted from caddy-cloudflare-ip to WEDOS IP ranges.

package caddy_wedos_ip

import (
	"bufio"
	"context"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

const (
	wedosIPsTxt = "https://ips.wedos.global/ips.txt"
)

func init() {
	caddy.RegisterModule(WedosIPRange{})
}

// WedosIPRange provides a range of IP address prefixes (CIDRs) retrieved from WEDOS Global.
type WedosIPRange struct {
	// refresh Interval
	Interval caddy.Duration `json:"interval,omitempty"`
	// request Timeout
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// Holds the parsed CIDR ranges from Ranges.
	ranges []netip.Prefix

	ctx  caddy.Context
	lock *sync.RWMutex
}

// CaddyModule returns the Caddy module information.
func (WedosIPRange) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.ip_sources.wedos",
		New: func() caddy.Module { return new(WedosIPRange) },
	}
}

// getContext returns a cancelable context, with a timeout if configured.
func (s *WedosIPRange) getContext() (context.Context, context.CancelFunc) {
	if s.Timeout > 0 {
		return context.WithTimeout(s.ctx, time.Duration(s.Timeout))
	}
	return context.WithCancel(s.ctx)
}

func (s *WedosIPRange) fetch(api string) ([]netip.Prefix, error) {
	ctx, cancel := s.getContext()
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	// WEDOS ips.txt can be space-separated, so scan tokens instead of lines.
	scanner.Split(bufio.ScanWords)

	var prefixes []netip.Prefix
	for scanner.Scan() {
		tok := scanner.Text()
		prefix, err := caddyhttp.CIDRExpressionToPrefix(tok)
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, prefix)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return prefixes, nil
}

func (s *WedosIPRange) getPrefixes() ([]netip.Prefix, error) {
	return s.fetch(wedosIPsTxt)
}

func (s *WedosIPRange) Provision(ctx caddy.Context) error {
	s.ctx = ctx
	s.lock = new(sync.RWMutex)

	// update in background
	go s.refreshLoop()
	return nil
}

func (s *WedosIPRange) refreshLoop() {
	if s.Interval == 0 {
		s.Interval = caddy.Duration(time.Hour)
	}

	ticker := time.NewTicker(time.Duration(s.Interval))
	// first time update
	s.lock.Lock()
	// it's nil anyway if there is an error
	s.ranges, _ = s.getPrefixes()
	s.lock.Unlock()
	for {
		select {
		case <-ticker.C:
			fullPrefixes, err := s.getPrefixes()
			if err != nil {
				break
			}

			s.lock.Lock()
			s.ranges = fullPrefixes
			s.lock.Unlock()
		case <-s.ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (s *WedosIPRange) GetIPRanges(_ *http.Request) []netip.Prefix {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ranges
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
//
//	wedos {
//	   interval val
//	   timeout val
//	}
func (m *WedosIPRange) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // Skip module name.

	// No same-line options are supported
	if d.NextArg() {
		return d.ArgErr()
	}

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		switch d.Val() {
		case "interval":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return err
			}
			m.Interval = caddy.Duration(val)
		case "timeout":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return err
			}
			m.Timeout = caddy.Duration(val)
		default:
			return d.ArgErr()
		}
	}

	return nil
}

// interface guards
var (
	_ caddy.Module            = (*WedosIPRange)(nil)
	_ caddy.Provisioner       = (*WedosIPRange)(nil)
	_ caddyfile.Unmarshaler   = (*WedosIPRange)(nil)
	_ caddyhttp.IPRangeSource = (*WedosIPRange)(nil)
)
