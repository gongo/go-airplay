package main

import (
	"os"
	"strconv"

	"github.com/gongo/go-airplay"
	"github.com/olekukonko/tablewriter"
)

func main() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "IP address", "Port"})

	for _, device := range airplay.Devices() {
		table.Append([]string{
			device.Name,
			device.Addr,
			strconv.Itoa(device.Port),
		})
	}

	table.Render()
}

// +----------------+-------------+------+
// |      NAME      | IP ADDRESS  | PORT |
// +----------------+-------------+------+
// | AppleTV.local. | 192.168.0.x | 7000 |
// +----------------+-------------+------+
