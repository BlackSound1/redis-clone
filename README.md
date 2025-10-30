# Redis Clone

This project is just a simple clone of a Redis server, written in Go.

Contains a `redis.conf` file to specify typical Redis settings.

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
- Strings: Strings are defined with a `+` and don't need to have a number associated with them because message will continue until newline bytes.
- Bulk strings: Bulk strings are defined with `$#\r\n`. They can have multiple words. The `#` defines the number of bytes in the string.
- Errors: Error messages are defined with `-`.
- Null: Can send a null message with a bulk string of length -1: `$-1\r\n`.
- Delete: Use `DEL` followed by a list of space-separated keys to delete. Returns number of keys actually deleted (not just attempted).
  If a given key doesn't exit, no-op, so it's safe to try deleting keys that don't exist.
- End a message: All RESP messages end with `\r\n`.

## Examples

```resp
+OK\r\n
```

A string-type message "`+`", with the content "OK", and that is the end of that part of the message.

```resp
-ERR error message 'foo'\r\n
```

An error-type message `"-"`, with the message `error message 'foo'`.

```resp
$6\r\nfoobar\r\n
```

A 6-byte bulk string-type message. The newline bytes must happen after the size, indicating the end of the size.
Then the message itself (must be correct number of bytes). Then the end of message bytes.

```resp
*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n
```

Which can be understood like:

```resp
*2\r\n
    $3\r\n
        foo\r\n
    $3\r\n
        bar\r\n
```

Redis sends messages as an array of bulk strings. This array has 2 bulk strings.
The first is a 3-byte string and the next is another 3-byte string.


## Persistence

Data in Redis can be stored persistently using AOF or RDB.

### RDB (Redis Database)

Takes keys and values in memory, converts them to bytes, and saves them all to DB file.

Can be automatic using `save` settings in the config file, or manual using a `SAVE` command.

#### Syntax of save settings

In the `redis.conf` file, automatic RDB saving can be configured using `save` commands. One such setting looks like `save 900 1`.
This means "save to the DB if 1 key changes in 900 seconds".

### AOF (Append Only File)

Creates a `.aof` file with RESP strings. Every once in a while, grabs all `SET` strings and appends them to the file.

To restore data, it takes all RESP strings from the file, parses them, and reruns all `SET` commands.
