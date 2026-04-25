package service

import (
	"net/netip"
)

func stringToNetIPPtr(s *string) *netip.Addr {
	if s == nil {
		return nil
	}

	ip, err := netip.ParseAddr(*s)

	if err != nil {
		return nil
	}
	return &ip
}
