# wink-mqtt in golang

[wink-mqtt](https://github.com/danielolson13/wink-mqtt/) equivalent written in golang.

I was using [wink-mqtt](https://github.com/danielolson13/wink-mqtt/) for a while and noticed some memory leak in the version on nodejs running on wink.

So I decided to create an equivalent native application. I landed on golang as the language to develop in as it has easy cross compilation and has a plethora of libraries available. This is my first golang application, so there may be places where best practices are not followed etc., but I just wanted something functional. Feel free to fix problems and submit PRs :)
My wink has been rock solid with this and my memory usage is very low:

```
[root@flex-dvt tmp]# free
total         used         free       shared      buffers
Mem:         60908        32780        28128            0            0
-/+ buffers:              32780        28128
Swap:            0            0            0
```

There is a JSON config file where you can specify MQTT broker connection params, topics etc. and also define your devices and attributes.
The config file path is specified when starting the application. The init script `mqtt` already has the correct command line. Currently it is assumed that both the binary and the config file are in `/opt/wink-mqtt`

## Build
To build, you will need to install golang. This is out of scope of this README. Then from the project directory:
```
GOOS=linux GOARCH=arm go build -o mqttwink
```

## Limitations
I only needed to interface with zwave locks (and only lock/unlock and battery levels) so that is what I have accounted for. One can easily add other types of devices and attributes.
