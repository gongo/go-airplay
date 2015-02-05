package airplay

// A Device is an AirPlay Device.
type Device struct {
	Name string
	Addr string
	Port int
}

// Devices returns all AirPlay devices in LAN.
func Devices() []Device {
	devices := []Device{}

	for _, entry := range searchEntry(&queryParam{}) {
		devices = append(
			devices,
			Device{
				Name: entry.hostName,
				Addr: entry.ipv4.String(),
				Port: int(entry.port),
			},
		)
	}

	return devices
}
