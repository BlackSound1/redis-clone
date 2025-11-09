# Redis Clone

This project is a minimal clone of a Redis server, written in Go.

Contains a `redis.conf` file to specify typical Redis settings.

# To Use

1. Make sure your normal Redis server (if present) is closed with `sudo systemctl stop redis-server.service`.
2. Launch this program with `go run .`.
3. Launch a Redis client with `redis-cli`.
4. Send commands via your Redis client to this server using RESP (see below).

# Commands

This minimal Redis server supports the following functionality:

- **Authenticate**: All commands except `COMMAND` and `AUTH` require authentication to use. Use `AUTH password` to log
  in. The password is defined in the `redis.conf` file. (See [Notes](#notes) for
  more).
- **Set or reset a key**: To create a new key or redefine an existing one, use `SET key value`.
- **Get a keys value**: To see what a key is set to, use `GET key`.
- **Delete**: Use `DEL key1 [key2...]` to delete one or more keys. Returns the number of keys actually deleted
  (not just tried to delete). If a given key doesn't exist, nothing happens, so it's safe to try deleting keys that don't exist.
- **Check for existence**: To check if a key exists in the DB, use `EXISTS key1 [key2...]`. Returns the number of given keys that exist, "0" if no such keys exist.
  - **Check for existence using pattern matching**: Use `KEYS pattern` to search all keys, given a pattern.
  Can search for an exact key by using the key name itself. Can search with any number of wildcard characters with `*`.
  Can search with exactly one wildcard character with `?`. Other possibilities exist. Returns a numbered list of keys similar to what you're looking for.
  <br>*For instance*: `KEYS n*ce` can find "nice" or "niece". Whereas `KEYS n?ce` would only find "nice", "nace", "nece", etc.
- **Save DB immediately**. Use `SAVE` to save immediately, ignoring RDB policy in the config. Typically, this isn't preferred in production, as it is blocking. `BGSAVE` is
  usually preferred instead. (See [Notes](#notes)).
  - **Save DB immediately (non-blocking)**: Use `BGSAVE` to immediately save the DB, ignoring RDB policy. This is *not* a true implementation of `BGSAVE` (See [Notes](#notes)).
- **Get how many keys are the in DB**: Use `DBSIZE` to see how many keys are stored in the DB.
- **Delete the whole DB**: To purge the entire DB, use `FLUSHDB`.
- **Set an expiry for a key**: Use `EXPIRE key seconds` to set an expiry for a key. Once the expiry time has passed,
  it deletes the key automatically. (See [Notes](#notes) for more).
- **Check how long a key has left to live**: Use `TTL key`. Returns:
  - "-2" if no such key is found.
  - "-1" if the key is found, but no expiry is found on it.
  - The number of seconds left to live if the expiring key is found.
- **Update the AOF file with the latest version of the DB**: The AOF file is just a list of SET ARRAYs that gets appended to. This
  can become outdated over time as the state of the in-memory DB changes. Use `BGREWRITEAOF` to
  rewrite the AOF file from scratch with only the current versions of each key.
- **Queue multiple commands to run atomically**: Use `MULTI` to start a transaction. This will activate a sort of
  'subshell' with "(TX)" added to the prompt. *You can't use `MULTI` within a
  transaction.* Input any commands you want and they will be "QUEUED".
  - Use `EXEC` to atomically run them all, their individual replies being output as a list. This will also exit the transaction.
  - Use `DISCARD` to leave the transaction without executing the commands.
- **Monitor other clients**: On a given client, use `MONITOR` to receive logs about other clients.
- **Get info about the server**: Use `INFO` to get server, client, memory, persistence, and general statistics.

# Config
Configuration is governed by the local `redis.conf` file. It supports the following settings:

**GENERAL**

These are intended to be settings that don't fit elsewhere.
- `dir folder`: Which `folder` to put AOF and RDB save data in.

**AOF**

AOF (Append Only File) settings. One of the ways to save the in-memory DB to a file.

- `appendonly`: Whether or not to support AOF saving. Possible values:
  - `yes`: Enable AOF file saving.
  - Anything else: Disable AOF file saving.
- `appendfilename file`: If `appendonly` is `yes`, what to call the file that gets saved to.
- `appendfsync`: How often to append to the AOF file. Possible values:
  - `always`: Always save the AOF file when a `SET` command is processed.
  - `everysec`: Save the AOF file every second.
  - `no`: Let the operating system decide when to save the AOF file.

**RDB**

RDB (Redis DataBase) settings. One of the ways to save the in-memory DB to a file.

- `save seconds numberOfKeys`: How many keys must change per given time interval to trigger saving to the RDB file. Can be set multiple times. *Example*: `save 2 1` means that 1 key must change in a 2 second window to trigger saving to RDB.
- `dbfilename file`: The name of the RDB file to save to.

**AUTH**

Authentication settings.

- `requirepass password`: The password that is required to authenticate a user. *Must authenticate before using most commands*.

**MEMORY**

Set how memory is handled.

- `maxmemory amount`: The maximum amount of the user's memory this program can use. Depending on eviction policy, memory may be automatically freed based on this setting.
  - If `amount` is `0`, no maximum is assumed.
  - Can store non-zero amounts in terms of `b`, `kb`, `mb`, or `gb`. 1 kb = 1024 b, etc. If no such modifier is given, the amount is interpreted in terms of bytes.
- `maxmemory-policy policy`: The eviction policy used to free memory if necessary. Possible policies:
  - `noeviction`: Do not try to evict keys if the maximum memory has been reached. Will throw an error if this is the case.
  - `allkeys-random`: Take a sample of keys and evict until there's enough memory left.
  - `allkeys-lru`: Similar to `allkeys-random`, but sort these random keys by which was least recently used.
  - `allkeys-lfu`: Similar to `allkeys-random`, but sort these random keys by which was least frequently used.
  - `volatile-random`: *Not implemented*.
  - `volatile-lru`: *Not implemented*.
  - `volatile-lfu`: *Not implemented*.
  - `volatile-ttl`: *Not implemented*.
- `maxmemory-samples numOfSamples`: For performance, only take `numOfSamples` keys from the DB to see if freeing them would satisfy the chosen eviction policy.

# An Overview of RESP

Redis messages are sent via a domain-specific language called RESP (REdis Serialization Protocol).

Every message sent in RESP will be an array filled with commands. The array must specify how many elements it contains.

- **End of message**: All RESP messages end with `\r\n`.
- **Arrays**: Defined using `*` and the number of elements in the array.
- **Strings**: Defined with a `+`. Don't have a number associated with them because the message will continue until the newline bytes. Contains only one word.
- **Bulk strings**: Defined with `$` and a number indicating the number of bytes in the string. They can have multiple words. 
- **Errors**: Error messages are defined with `-`. Must be written with `ERR [text...]`.
- **Null**: Can send a null message as a bulk string with negative length: `$-1\r\n`.

## Examples

**String**

```resp
+OK\r\n
```

A string-type message "`+`", with the content "OK".

**Error**

```resp
-ERR error message 'foo'\r\n
```

An error-type message "`-`", with the message `error message 'foo'`. Note the mandatory `ERR`.

**Bulk**

```resp
$6\r\nfoobar\r\n
```

A 6-byte bulk-type message. The newline bytes must happen after the size, indicating the end of the size portion.
Then the message itself (must have the correct number of bytes).

**Array**

```resp
*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n
```

An array-type message indicating 2 commands. It can be understood like:

```resp
*2\r\n           // 2 elements in the array
    $3\r\n       // A 3-byte bulk string
        foo\r\n  // The string
    $3\r\n       // A 3-byte bulk string
        bar\r\n  // The string
```

# Persistence

Data in Redis can be stored persistently to files using AOF and/ or RDB.

**RDB (Redis DataBase)**

Takes the keys and values in memory, converts them to bytes, and saves them all to a file.

Can be automatic using `save` settings in the config file, or manual using a `SAVE` command.

**AOF (Append Only File)**

Creates a file with RESP strings. Every once in a while, grabs all `SET` strings and appends them to the file.

To restore data, it takes all `SET` commands from the file, parses them, and reruns them all.

# Notes

`BGSAVE` uses `COW (Copy On Write)` in the actual Redis implementation. This uses an OS-provided memory optimization algorithm. This isn't quite possible in Go, because Go is garbage collected. So true background saving is not supported.

Saving RDB files has SHA256 checksum protection to ensure data is saved correctly.

Real Redis has a much more complicated way of authenticating users called ACL (Access Control List). This approach is much simpler, requiring only a password.

Real Redis handles expiry in 2 ways: using `GET`, and other commands to see if a key is expired, or by using a background process to periodically check a sample of keys for expiry.
This second method is not implemented here, and may never be.
