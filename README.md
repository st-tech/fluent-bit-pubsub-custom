# fluent-bit output plugin for google pubsub

<p align="left">    

  <a href="https://hits.seeyoufarm.com"/><img src="https://hits.seeyoufarm.com/api/count/incr/badge.svg?url=https%3A%2F%2Fgithub.com%2Fgjbae1212%2Ffluent-bit-pubsub"/></a>
  <a href="/LICENSE"><img src="https://img.shields.io/badge/license-MIT-GREEN.svg" alt="license" /></a>
  <a href="https://goreportcard.com/report/github.com/gjbae1212/fluent-bit-pubsub"><img src="https://goreportcard.com/badge/github.com/gjbae1212/fluent-bit-pubsub" alt="Go Report Card" /></a>

</p>

This plugin is used to publish data to queue in google pubsub. 

You could easily use it.

## Build
A bin directory already has been made binaries for mac, linux.

If you should directly make binaries for mac, linux
```bash
# local machine binary
$ make local-build

# Your machine is mac, and if you should do to retry cross compiling for linux.
# A command in below is required a docker.  
$ make build-linux
```

## Usage
### configuration options for fluent-bit.conf
| Key           | Description                                    | Default        |
| ----------------|------------------------------------------------|----------------|
| Project         | Google Cloud project ID | NONE(required) |
| Topic           | Google Cloud Pub/Sub topic name | NONE(required) |
| Format          | The type of message to be sent to pubsub. Currently, only `json` is supported. | NONE(optional) |
| Attributes      | JSON string specifying message attributes | NONE(optional) |
| Debug           | Print debug log | False(optional) |
| Timeout         | The maximum time that the client will attempt to publish a bundle of messages. (millsecond) | 60000(optional)|
| DelayThreshold  | Publish a non-empty batch after this delay has passed. (millsecond) | 1  |
| ByteThreshold   | Publish a batch when its size in bytes reaches this value. | 1000000(optional) |
| CountThreshold  | Publish a batch when it has been reached count of messages. | 100(optional) |
| BufferedByteLimit| The maximum number of bytes that the client will buffer before the messages are sent to Pub/Sub.(byte) | 10000000(optional)|


### Example fluent-bit.conf
```conf
[Output]
    Name pubsub
    Match *
    Project your-project(custom)
    Topic your-topic-name(custom)
    Format json
    Attributes {"key1":"value1","key2":"value2"} 
```

### Example exec
```bash
$ fluent-bit -c [your config file] -e pubsub.so 
```
