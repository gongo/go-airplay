package airplay

// discovery.go was created in reference to github.com/armon/mdns/client.go

import (
	"log"
	"net"
	"time"

	"github.com/miekg/dns"
)

const (
	mdnsAddr = "224.0.0.251"
	mdnsPort = 5353
)

var (
	mdnsUDPAddr = &net.UDPAddr{
		IP:   net.ParseIP(mdnsAddr),
		Port: mdnsPort,
	}
)

type discovery struct {
	conn     *net.UDPConn
	closed   bool
	closedCh chan int
}

type entry struct {
	ipv4       net.IP
	ipv6       net.IP
	port       int
	hostName   string
	domainName string
}

type queryParam struct {
	timeout  time.Duration
	maxCount int
}

func newDiscovery() (*discovery, error) {
	conn, err := net.ListenMulticastUDP("udp4", nil, mdnsUDPAddr)
	if err != nil {
		return nil, err
	}

	d := &discovery{
		conn:     conn,
		closed:   false,
		closedCh: make(chan int),
	}
	return d, nil
}

func searchEntry(params *queryParam) []*entry {
	d, _ := newDiscovery()
	defer d.close()

	if params.timeout == 0 {
		params.timeout = 1 * time.Second
	}

	if params.maxCount == 0 {
		params.maxCount = 5
	}

	return d.query(params)
}

func (d *discovery) query(params *queryParam) []*entry {
	// Send question
	m := new(dns.Msg)
	m.SetQuestion("_airplay._tcp.local.", dns.TypePTR)
	buf, _ := m.Pack()
	d.conn.WriteToUDP(buf, mdnsUDPAddr)

	msgCh := make(chan *dns.Msg, 4)
	go d.receive(msgCh)

	entries := []*entry{}
	finish := time.After(params.timeout)

L:
	for {
		select {
		case response := <-msgCh:
			entry := parse(response)
			if entry == nil {
				continue
			}
			entries = append(entries, entry)

			if len(entries) >= params.maxCount {
				break L
			}
		case <-finish:
			break L
		}
	}

	return entries
}

func (d *discovery) close() {
	d.closed = true
	close(d.closedCh)
	d.conn.Close()
}

func (d *discovery) receive(ch chan *dns.Msg) {
	buf := make([]byte, dns.DefaultMsgSize)

	for !d.closed {
		n, _, err := d.conn.ReadFromUDP(buf)
		if err != nil {
			log.Println(err)
			continue
		}
		msg := new(dns.Msg)

		if err := msg.Unpack(buf[:n]); err != nil {
			log.Println(err)
			continue
		}

		select {
		case ch <- msg:
		case <-d.closedCh:
			return
		}
	}
}

func parse(resp *dns.Msg) *entry {
	entry := new(entry)

	for _, answer := range resp.Answer {
		switch rr := answer.(type) {
		case *dns.PTR:
			entry.domainName = rr.Ptr
		}
	}

	if entry.domainName == "" {
		log.Println("airplay: [ERR] Failed to get PTR record")
		return nil
	}

	for _, extra := range resp.Extra {
		switch rr := extra.(type) {
		case *dns.SRV:
			if rr.Hdr.Name == entry.domainName {
				entry.hostName = rr.Target
				entry.port = int(rr.Port)
			}
		}
	}

	if entry.hostName == "" {
		log.Println("airplay: [ERR] Failed to get SRV record")
		return nil
	}

	for _, extra := range resp.Extra {
		switch rr := extra.(type) {
		case *dns.A:
			if rr.Hdr.Name == entry.hostName {
				entry.ipv4 = rr.A
			}
		case *dns.AAAA:
			if rr.Hdr.Name == entry.hostName {
				entry.ipv6 = rr.AAAA
			}
		}
	}

	return entry
}
