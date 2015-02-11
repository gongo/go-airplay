# Example code for `airplay.Devices()`

## Install

    $ go get github.com/gongo/go-airplay/example/devices

or

    $ cd $GOPATH/src/github.com/gongo/go-airplay/example/devices
    $ go build

## Run

    $ devices

```
+------------------+------------+------+
|       NAME       | IP ADDRESS | PORT |
+------------------+------------+------+
| AppleTV.local.   | 192.0.2.1  | 7000 |
| AirServer.local. | 192.0.2.2  | 7000 |
+------------------+------------+------+

* (AppleTV.local.)
  Model Name         : AppleTV2,1
  MAC Address        : FF:FF:FF:FF:FF:FF
  Server Version     : 222.22
  Features           : 0xFFFFFFF,0xF
  Password Required? : no

* (AirServer.local.)
  Model Name         : AppleTV3,2
  MAC Address        : 00:00:00:00:00:00
  Server Version     : 111.11
  Features           : 0x10000000
  Password Required? : yes
```
