/*
File Name:  Network Detection.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import "net"

// FindInterfaceByIP finds an interface based on the IP. The IP must be available at the interface.
func FindInterfaceByIP(ip net.IP) (iface *net.Interface, ipnet *net.IPNet) {
	interfaceList, err := net.Interfaces()
	if err != nil {
		return nil, nil
	}

	// iterate through all interfaces
	for _, ifaceSingle := range interfaceList {
		addresses, err := ifaceSingle.Addrs()
		if err != nil {
			continue
		}

		// iterate through all IPs of the interfaces
		for _, address := range addresses {
			addressIP := address.(*net.IPNet).IP

			if addressIP.Equal(ip) {
				return &ifaceSingle, address.(*net.IPNet)
			}
		}
	}

	return nil, nil
}

// NetworkListIPs returns a list of all IPs
func NetworkListIPs() (IPs []net.IP, err error) {

	interfaceList, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// iterate through all interfaces
	for _, ifaceSingle := range interfaceList {
		addresses, err := ifaceSingle.Addrs()
		if err != nil {
			continue
		}

		// iterate through all IPs of the interfaces
		for _, address := range addresses {
			addressIP := address.(*net.IPNet).IP
			IPs = append(IPs, addressIP)
		}
	}

	return IPs, nil
}

// NetworkListOutput provides a user friendly output
func NetworkListOutput() (text string) {

	interfaceList, err := net.Interfaces()
	if err != nil {
		return "Error " + err.Error()
	}

	// iterate through all interfaces
	for _, ifaceSingle := range interfaceList {
		text += "Interface " + ifaceSingle.Name + ":\n"
		text += "  MAC:        " + ifaceSingle.HardwareAddr.String() + "\n"

		addresses, err := ifaceSingle.Addrs()
		if err != nil {
			text += "  Error getting addresses: " + err.Error() + "\n\n"
			continue
		}

		// iterate through all IPs of the interfaces
		for _, address := range addresses {
			text += "  IP:         " + address.(*net.IPNet).IP.String() + "\n"
		}

		addresses2, err := ifaceSingle.MulticastAddrs()
		if err != nil {
			text += "  Error getting multicast addresses: " + err.Error() + "\n\n"
			continue
		}

		for _, address := range addresses2 {
			text += "  Multicast:  " + address.(*net.IPAddr).IP.String() + "\n"
		}

		text += "\n"
	}

	return text
}

// IsIPv4 checks if an IP address is IPv4
func IsIPv4(IP net.IP) bool {
	return IP.To4() != nil
}

// IsIPv6 checks if an IP address is IPv6
func IsIPv6(IP net.IP) bool {
	return IP.To4() == nil && IP.To16() != nil
}

// TODO: Incorporate functions ReachabilityTo and Routable from:
// https://github.com/tendermint/go-p2p/blob/master/netaddress.go
