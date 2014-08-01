package airplay

import "github.com/armon/mdns"

// A Device is an AirPlay Device.
type Device struct {
	Name string
	Addr string
	Port int
}

// Devices returns all AirPlay devices in LAN.
func Devices() []Device {
	devices := []Device{}
	entriesCh := make(chan *mdns.ServiceEntry, 4)
	defer close(entriesCh)

	go func() {
		for entry := range entriesCh {
			devices = append(
				devices,
				Device{
					Name: entry.Name,
					Addr: entry.Addr.String(),
					Port: entry.Port,
				},
			)
		}
	}()

	mdns.Lookup("_airplay._tcp", entriesCh)

	return devices
}
