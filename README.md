# gominer
GPU miner for sia in go

All available opencl capable GPU's are detected and used in parallel.

## Binary releases

[Binaries for Windows and Linux are available in the corresponding releases](https://github.com/robvanmieghem/gominer/releases)


## Installation from source

### Prerequisites
* go version 1.4.2 or above (earlier version might work or not), check with `go version`
* opencl libraries on the library path
* gcc

```
go get github.com/robvanmieghem/gominer
```

## Run
```
gominer
```

Usage:
```
  -url string
    	siad host and port (default "localhost:9980")
        for stratum servers, use `stratum+tcp://<host>:<port>`
  -user string
        username, most stratum servers take this in the form [payoutaddress].[rigname]
        This is optional, if solo mining sia, this is not needed
  -I int
    	Intensity (default 28)
  -E string
        Exclude GPU's: comma separated list of devicenumbers
  -cpu
    	If set, also use the CPU for mining, only GPU's are used by default
  -v	Show version and exit
```

See what intensity gives you the best hashrate, increasing the intensity also increases the stale rate though.

## Stratum support

Stratum support is implemented as defined on https://siamining.com/stratum

## Developer fee

A developer fee of 1% is created by submitting 1% of the shares for my address if using the stratum protocol. The code is open source so you can simply remove that line if you want to. To make it easy for you, the exact line is https://github.com/robvanmieghem/gominer/blob/master/clients/siastratum.go#L359 if you do not want to support the gominer development.

## FAQ

- ERROR fetching work - Get http://localhost:9980/miner/headerforwork: dial tcp 127.0.0.1:9980: connection refused

  Make sure `siad` is running

- What is `siad`?

  Check the sia documentation

- I don't know how to set up siad or the sia UI wallet, how do I do that?

  Check the sia documentation.

- You have to help me set up mining SIA

  No I don't

- Can you log in on my machine to configure my mining setup?

  No

- I don't know how to get it working, can you help me?

  I get this question at least once a day. Seriously, you can not expect me to set up and support everyone's mining equipment.

- I don't know how to get it working, can you help me please please please ?

  Everyone has his price, make me an offer I can't refuse so I don't have to continue answering `no` to this question.

## Support development

If you really want to, you can support the gominer development:

SIA: 79b9089439218734192db7016f07dc5a0e2a95e873992dd782a1e1306b2c44e116e1d8ded910

BTC: 36bQun9zTTvBzHVxvokrLeBACEPGQrebAP
