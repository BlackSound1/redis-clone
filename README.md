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

- **Arrays**: Arrays are defined using `*#`, where `#` is the number of elements in the array.
- **Strings**: Strings are defined with a `+` and don't need to have a number associated with them because message will continue until newline bytes.
- **Bulk strings**: Bulk strings are defined with `$#\r\n`. They can have multiple words. The `#` defines the number of bytes in the string.
- **Errors**: Error messages are defined with `-`.
- **Null**: Can send a null message with a bulk string of length -1: `$-1\r\n`.
- **Delete**: Use `DEL key1 [key2...]` to delete 1 or more keys. Returns number of keys actually deleted (not just attempted).
  If a given key doesn't exit, no-op, so it's safe to try deleting keys that don't exist.
- **Check for existence**: To check if a key exists in the DB use `EXISTS key1 [key2...]`. Can also take multiple space-separated keys.
  - **Check for existence using pattern matching**: Use `KEYS pattern` to search all keys, given a pattern.
  Can search for a key exactly by using the key name, with any number of wildcard characters with `*`, or with exactly 1 wildcard character with `?`. Other possibilities exist.
- **Save DB immediately**. Sometimes you may want to save the DB immediately to a file regardless of your AOF or RDB policies. To do so, use `SAVE`, which takes no arguments.
  Typically, this isn't preferred in production, as it is blocking. `BGSAVE` is usually preferred instead. (See [Notes](#notes)).
- **Get how many keys are the in DB**: Use `DBSIZE` with no arguments to get how many keys are stored in the DB.
- **Delete the whole DB**: To purge the whole DB, use `FLUSHDB` with no arguments.
- **Authenticate**: All commands except `COMMAND` and `AUTH` require authentication to use. Use `AUTH <PASS>` to log in. By default, the password is `hey`, but this can be changed in the `redis.conf` file. (See [Notes](#notes) for more).
- **End a message**: All RESP messages end with `\r\n`.

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

## Notes

`BGSAVE` uses `COW (Copy-On-Write)` in the actual Redis implementation. This uses an OS-provided memory optimization algorithm.
This isn't quite possible in Go, because Go is garbage collected.
So true background saving is not supported.

Saving RDB files has SHA256 checksum protection to ensure data is saved correctly.

Real Redis has a much more complicated way of authenticating users called ACL (Access Control List).
The implemented approach is much simpler. ACL involves a username and password, but this implementation only supports a password.
