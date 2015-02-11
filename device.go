package airplay

// A Device is an AirPlay Device.
type Device struct {
	Name  string
	Addr  string
	Port  int
	Extra DeviceExtra
}

// A DeviceExtra is extra information of AirPlay device.
type DeviceExtra struct {
	Model              string
	Features           string
	MacAddress         string
	ServerVersion      string
	IsPasswordRequired bool
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

// FirstDevice return the first found AirPlay device in LAN.
func FirstDevice() Device {
	params := &queryParam{maxCount: 1}

	for _, entry := range searchEntry(params) {
		return entryToDevice(entry)
	}

	return Device{}
}

func entryToDevice(entry *entry) Device {
	extra := DeviceExtra{
		Model:              entry.textRecords["model"],
		Features:           entry.textRecords["features"],
		MacAddress:         entry.textRecords["deviceid"],
		ServerVersion:      entry.textRecords["srcvers"],
		IsPasswordRequired: false,
	}

	if pw, ok := entry.textRecords["pw"]; ok && pw == "1" {
		extra.IsPasswordRequired = true
	}

	return Device{
		Name:  entry.hostName,
		Addr:  entry.ipv4.String(),
		Port:  int(entry.port),
		Extra: extra,
	}
}
