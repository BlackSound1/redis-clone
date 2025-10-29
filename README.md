# Redis Clone

This project is just a simple clone of a Redis server, written on Go.

## To Use

1. Make sure your normal Redis server is closed with `sudo systemctl stop redis`.
2. Launch this program with `go run .`.
3. Launch your local Redis client with `redis-cli`.
4. Send RESP messages via your Redis client to this server using RESP (see below).

## RESP

Redis messages are sent via a domain-specific language called RESP (REdis Serialization Protocol).

Every message sent in RESP will be an array filled with commands.

Overview of RESP:

- Arrays: Arrays are defined using `*#`, where `#` is the number of elements in the array.
- Strings: Strings are defined with a `+` and don't need to have a number associated with them. Strings are 1 word.
- Bulk strings: Bulk strings are defined with `$#`. They can have multiple words. The `#` defines the number of bytes in the string.
- End a message: All RESP messages end with `\r\n`.
