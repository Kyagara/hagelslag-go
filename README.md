# hagelslag

cool scanner <(= w =)>

## Why

This project started as a [C](https://github.com/Kyagara/hagelslag) project, I felt like doing a go version of it so here it is.

The `hagelslag` name came from a friend, well, eating a [Hagelslag](https://en.wikipedia.org/wiki/Hagelslag).

## How

`hagelslag` works by generating all possible IP addresses, all 4.3 billion of them, in a loop, checking if they are [reserved](https://en.wikipedia.org/wiki/Reserved_IP_addresses) and sending them to the `ips` channel.

Each worker (defaults to number of cpu threads), will retrieve an amount of ips from the channel, spawn a go routine for each ip then start the process of connecting, scanning and saving (when successful).

### Scanning

> Requests that will be made.

- `http`: send a `GET` request.

- `minecraft`: send a handshake, status request packet.

- `veloren`: send a init packet, server info packet.

### Saving

Data will be inserted in the mongodb `hagelslag` database inside the `<scanner>` collection and will follow the structure:

```json
{
    // ip address
    "_id": "<ip>",
    // time in milliseconds
    "latency": 0,
    // data can be a string or json
    "data": ""
}
```

## Installing

```
go install github.com/Kyagara/hagelslag-go@latest
```

## Ideas

- The amount of workers/tasks running might not be good, find a better default based on some values.

- Add more checks for response size, also make a better read function for HTTP, currently it reads until EOF, maybe add a limit.

- Maybe add bedrocks servers to the Minecraft scanner.

- Remove this weird virus that keeps adding Frieren in the code.
