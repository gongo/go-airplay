package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gongo/go-airplay"
	"github.com/olekukonko/tablewriter"
)

func main() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "IP address", "Port"})

	extraTemplate := `
* (%s)
  Model Name         : %s
  MAC Address        : %s
  Server Version     : %s
  Features           : %s
  Password Required? : %s
`
	extra := ""

	for _, device := range airplay.Devices() {
		table.Append([]string{
			device.Name,
			device.Addr,
			strconv.Itoa(int(device.Port)),
		})

		passwordRequiredFlag := "no"
		if device.Extra.IsPasswordRequired {
			passwordRequiredFlag = "yes"
		}

		extra += fmt.Sprintf(
			extraTemplate,
			device.Name,
			device.Extra.Model,
			device.Extra.MacAddress,
			device.Extra.ServerVersion,
			device.Extra.Features,
			passwordRequiredFlag,
		)
	}

	table.Render()
	fmt.Println(extra)
}
