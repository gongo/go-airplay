package airplay

// discovery.go was created in reference to github.com/armon/mdns/client.go

import (
	"fmt"
	"log"
	"net"
	"strings"
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
	searchDomain = "_airplay._tcp.local."
)

type discovery struct {
	mconn    *net.UDPConn
	uconn    *net.UDPConn
	closed   bool
	closedCh chan int
}

type entry struct {
	ipv4        net.IP
	port        int
	hostName    string
	domainName  string
	textRecords map[string]string
}

type queryParam struct {
	timeout  time.Duration
	maxCount int
}

func newDiscovery() (*discovery, error) {
	mconn, err := net.ListenMulticastUDP("udp4", nil, mdnsUDPAddr)
	if err != nil {
		return nil, err
	}

	uconn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}

	d := &discovery{
		mconn:    mconn,
		uconn:    uconn,
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

	if params.maxCount <= 0 {
		params.maxCount = 5
	}

	return d.query(params)
}

func (d *discovery) query(params *queryParam) []*entry {
	// Send question
	m := new(dns.Msg)
	m.SetQuestion(searchDomain, dns.TypePTR)
	buf, _ := m.Pack()
	d.uconn.WriteToUDP(buf, mdnsUDPAddr)

	msgCh := make(chan *dns.Msg, 8)
	go d.receive(d.uconn, msgCh)
	go d.receive(d.mconn, msgCh)

	entries := []*entry{}
	finish := time.After(params.timeout)

L:
	for {
		select {
		case response := <-msgCh:
			// Ignore question message
			if !response.MsgHdr.Response {
				continue
			}

			entry, err := d.parse(response)
			if err != nil {
				fmt.Println(err)
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
	d.uconn.Close()
	d.mconn.Close()
}

func (d *discovery) receive(l *net.UDPConn, ch chan *dns.Msg) {
	buf := make([]byte, dns.DefaultMsgSize)

	for !d.closed {
		n, _, err := l.ReadFromUDP(buf)
		if err != nil {
			// Ignore error that was occurred by Close() while blocked to read packet
			if !d.closed {
				log.Printf("airplay: [ERR] Failed to receive packet: %v", err)
			}
			continue
		}

		msg := new(dns.Msg)
		if err := msg.Unpack(buf[:n]); err != nil {
			log.Printf("airplay: [ERR] Failed to unpack packet: %v", err)
			continue
		}

		select {
		case ch <- msg:
		case <-d.closedCh:
			return
		}
	}
}

func (d *discovery) parse(resp *dns.Msg) (*entry, error) {
	entry := &entry{textRecords: make(map[string]string)}

	for _, answer := range resp.Answer {
		switch rr := answer.(type) {
		case *dns.PTR:
			entry.domainName = rr.Ptr
		}
	}

	if entry.domainName == "" {
		return nil, NewDNSResponseParseError("PTR", resp.Answer)
	}

	for _, extra := range resp.Extra {
		switch rr := extra.(type) {
		case *dns.SRV:
			if rr.Hdr.Name == entry.domainName {
				entry.hostName = rr.Target
				entry.port = int(rr.Port)
			}
		case *dns.TXT:
			if rr.Hdr.Name == entry.domainName {
				for _, txt := range rr.Txt {
					lines := strings.Split(txt, "=")
					entry.textRecords[lines[0]] = lines[1]
				}
			}
		}

	}

	if entry.hostName == "" {
		return nil, NewDNSResponseParseError("SRV", resp.Extra)
	}

	for _, extra := range resp.Extra {
		switch rr := extra.(type) {
		case *dns.A:
			if rr.Hdr.Name == entry.hostName {
				entry.ipv4 = rr.A
			}
		}
	}

	if entry.ipv4.String() == "<nil>" {
		return nil, NewDNSResponseParseError("A", resp.Extra)
	}

	return entry, nil
}
