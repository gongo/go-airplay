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
			entryToDevice(entry),
		)
	}

	return devices
}

// FirstDevice return the first AirPlay device in LAN it is found.
func FirstDevice() Device {
	params := &queryParam{maxCount: 1}

	for _, entry := range searchEntry(params) {
		return entryToDevice(entry)
	}

	return Device{}
}

func entryToDevice(entry *entry) Device {
	return Device{
		Name: entry.hostName,
		Addr: entry.ipv4.String(),
		Port: int(entry.port),
	}
}
