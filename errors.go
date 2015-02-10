package airplay

import (
	"fmt"

	"github.com/miekg/dns"
)

type DNSResponseParseError struct {
	Type    string
	Records []dns.RR
}

func NewDNSResponseParseError(t string, r []dns.RR) *DNSResponseParseError {
	return &DNSResponseParseError{
		Type:    t,
		Records: r,
	}
}

func (e *DNSResponseParseError) Error() string {
	return fmt.Sprintf(
		"airplay: [ERR] Failed to get %s record:\n%s",
		e.Type,
		e.Records,
	)
}
