# hagelslag

cool scanner <(= w =)>

## Why

Starting as a C [project](https://github.com/Kyagara/hagelslag), I felt like doing a go version of it.

The `hagelslag` name came from a friend, well, eating a [Hagelslag](https://en.wikipedia.org/wiki/Hagelslag).

## How

`hagelslag` works by generating all possible IPv4 addresses, checking if they are [reserved](https://en.wikipedia.org/wiki/Reserved_IP_addresses) or not and sending them to workers.

Each worker will wait for addresses coming from a channel, spawn a go routine for each address then start the process of connecting, scanning and saving (when successful).

### CLI

```bash
-ip
    IP address to start from, without port
-scanner
    Scanner to use (default: http)
-port
    Override the scanners port
-uri    
    MongoDB URI (default: mongodb://localhost:27017)
-only-connect
    Skip scanning, connect and save if successful (default: false)
-rate
    Limit of connections, be careful with this value (default: 1000)
```

### Scanning

If not set to `OnlyConnect`, the scanner will do the following:

- `http`: send a `GET` request.

- `minecraft`: send a handshake, status request packet.

- `veloren`: send a init packet, server info packet.

Current behaviour is to read until the response reaches the limit of 15Mb or EOF is encountered.

### Saving

Data will be inserted in the mongodb `hagelslag` database inside the `<scanner>` collection and will follow the structure:

> mongodb has a limit of 16Mb for a document, if a response exceeds 15Mb, the json/html that will be saved _will_ be malformed, validate the data before using it.

> `data` field can be a string (for html or malformed response) or a json object.

```json
{
    "_id": "<address>",
    "latency": 0,
    "data": ""
}
```

## TODO/Ideas

- Maybe an interface for both DialTCP and DialUDP.

- Improve logging.

- At high rates, DB connection errors out, maybe using bulk write can solve this.

- Maybe add bedrocks servers to the Minecraft scanner.

- Remove this weird virus that keeps adding Frieren in the code.
