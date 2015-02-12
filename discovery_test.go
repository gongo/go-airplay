package airplay

import (
	"log"
	"net"
	"testing"

	"github.com/miekg/dns"
)

func rr(s string) dns.RR {
	rr, err := dns.NewRR(s)
	if err != nil {
		log.Fatal(err)
	}
	return rr
}

func TestParse(t *testing.T) {
	m := new(dns.Msg)
	m.Answer = []dns.RR{
		rr("example.com. 10 IN PTR GongoTV.example.com."),
	}
	m.Extra = []dns.RR{
		rr("GongoTV.local. 10 IN A    192.0.2.1"),
		rr("GongoTV.example.com. 120 IN SRV 0 0 7000 GongoTV.local."),
		rr("GongoTV.example.com. 120 IN TXT \"deviceid=00:00:00:00:00:00\" \"model=AppleTV2,1\""),
	}

	d, _ := newDiscovery()
	entry, err := d.parse(m)

	if err != nil {
		t.Fatalf("Unexpected error (%v)", err)
	}

	if !entry.ipv4.Equal(net.ParseIP("192.0.2.1")) {
		t.Errorf("Unexpected entry.ipv4 (%s)", entry.ipv4)
	}

	if entry.port != 7000 {
		t.Errorf("Unexpected entry.port (%d)", entry.port)
	}

	if len(entry.textRecords) != 2 {
		t.Errorf("Unexpected entry.textRecords (%v)", entry.textRecords)
	}
}

func TestParseErrorWithoutRequireRecords(t *testing.T) {
	m := new(dns.Msg)
	d, _ := newDiscovery()

	if _, err := d.parse(m); err == nil {
		t.Fatal("It should occurs [PTR not found] error")
	}

	m.Answer = []dns.RR{
		rr("example.com. 10 IN PTR GongoTV.example.com."),
	}

	if _, err := d.parse(m); err == nil {
		t.Fatal("It should occurs [SRV not found] error")
	}

	m.Extra = []dns.RR{
		rr("GongoTV.example.com. 120 IN SRV 0 0 7000 GongoTV.local."),
	}

	if _, err := d.parse(m); err == nil {
		t.Fatal("It should occurs [A not found] error")
	}
}
