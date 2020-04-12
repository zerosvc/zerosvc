# zerosvc

Queue-oriented (AMQP for now, ZMQ someday) service/RPC/event framework

## Implementations

* [Golang](https://github.com/zerosvc/go-zerosvc) [![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/zerosvc/go-zerosvc)


## Concepts

### Paths

Paths are in `/` - separated format. 

### Services
#### Service providers

( shortened to **SP** )

Service providers provide service to components of the system.

Service provider **have** to ACK or NACK task if client sends appropriate (mapped to "reply to" in the transports that support it)
If SP generates output it will send it to path provided in request. 
If request does not provide necessary information (e.g. invalid request) it **must** NAK it. 
If client wants to discard output it **must** not provide "reply to"
Service **may** send more than one message to ReplyTo header (e.g. start and end of a job)

SP have **name** and optional one or more instances and sub-instances, separated by dot, 
e.g. notification service could have name `notify` and instances `notify/xmpp`, `notify/email`, `notify/sms`, 
with sub instances `notify/sms/prio` and `notify/sms/bulk`.

#### Instances

Instances are for dividing services of same type into logical instances.

For example `service/nmap` might be divided into `service/nmap/dc1` and `service/nmap/dc2` to give client ability to choose a location without sending request to specific node

Or job manager could have `service/jobs/bulk` and `service/jobs/important` to divide work between slower and faster workers

Note that how that works depends on transport. MQTTv3 does not have support for shared subscriptions

It can be also used to separate resources of different projects and then futher subdivide it inside a given project for example multiple projects with multiple varnish cache pools: `service.cache-invalidator.project1.varnish-gfx`, `service.cache-invalidator.project2.varnish-mm`


#### Accepting a task

SP **must** generate name of the task when accepting, it **must** be unique

task **should** be named based on client node name and uniquely generated ID, for example

`web1_example_com-parser.1382439c-d46b-425f-88a1-98e66d462da1`

If client sends `on-start` header with return path, SP **must** send an ACK with task ID to that path, or NAK if job is invalid.

If client sends `on-finish` header with return path, SP **may** send an ACK when job is finished.

If client sends `on-finish` header with return path and SP does not support sending `on-finish` ACK, it **must** respond with `no-on-finish: 1` in `on-start` ACK to inform client it wont be sending `on-finish` even if it was requested

If client sends `output-to` header with path and SP task returns output it **must** send it to provided addres

If client sends `output-to` header with path and SP task does not return output, it **must** NAK it as that would break tasks that use that to forward job to other component

If client sends `maybe-output-to` header with path, SP **must** send output to path only if task generates output

If client sends `maybe-output-to` and `output-to` in same message it **must** be NAK.

### Nodes

There are 2 types of nodes:

#### Persistent

Emit heartbeat and are discoverable via discovery.* namespace. Can provide service.

#### Transient

Nodes not registered via discovery service, they can generate/consume events and use existing services but not provide them

### Protocol

All data types are same as in JSON

Message is divided in 3 parts:

* Routing key - used in low-level routing and rough qualification of message type
* Header - passing basic parameters, and in advanced filtering
* Body - container for service requests/responses

#### Routing key

String of 255 ascii chars used to address components of the system. It is formatted by parts separated by dots.
First part is a type of [endpoint](doc/endpoints.md)
Second part **must** be a name of underlying service and each subsequent part **should** further subdivide service as in [Instances](#instances)

So for example if we need to address 6 clusters of service distibuted into 2 DCs, routing key should look like `service.batch-jobs.dc1.cluster3`

#### Filtering

each dotted part of routing key can be filtered using 2 filters:

* `+`(plus) - means "substiture one word"
* `#` (hash) - means "substitute more than one word"

Transport should translate that mapping to native

so `service/img/+/png` will match `service/img/crop/png` but not `service/img/png` or `service/img/dc1/crop/png`

and `service/#/png` will match all of those

#### Routing

Routing engine **must** be able to route using prefixes (so `prefix.#` filter equivalent) and **must** be able to filter (at minimum, receive and ignore not matching) via it

### Header

A set of key=>value pairs with key being 250 ascii string and value is binary limited by header size.

Minimum header size is 64KiB of json-encoded data (even if underlying transport have to use less/more data to accomplish it).

It should be encoded on transport level (so client only passes a hash of data).

Transport **may** add new headers but **must not** modify passed ones. If transport need those headers for it's own use it should prefix received headers on send, and remove that prefix on receive.

Transport **can** add keys on receive but they **must** be contained under `_transport-` prefix

### Signatures

Header for containing signatures is **must** be called `_sec-sig`. 

#### Binary format

| byte | format  | usage  |
| --- | ---  |  --- |
| 0 | uint8 | type |
| 1 | uint8 | length of signature |
| 2.. | 1-255 bytes|  |

#### Encoding

Sig block should be encoded in base64 

#### Signature types

| ID | name    | description |
| -- | ---     | ---         |
| 0  | invalid | invalid type |
| 1  | Ed25519 | raw Ed25519 signature |
| 2  | X509    | X509  CA-based sig |

#### required keys

* `node-name` - 1-256 byte UTF-8 node name. Human readable, preferably in `fqdn@appname:instance` form
* `node-uuid` - 32 byte node UUID - in case of persistent nodes it should be generated at first start and saved, especially if your application stores data with node
* `ts` - unixtime, can be s/ms/us accuracy e.g `123456.001`
* `sha256` - checksum of data part

### reserved keys prefix

* `_transport-` - transport specific info (like used auth/id)
* `_sec-` - security-related parts of protocol like signature **should not** be generated/verified by client/service directly but by library handling the communication

## Body

Body **should** be json encoded message. Encoding **must** be done before put into transport, transport **must not** encode it and should treat it as binary blob. Blob **must** be checksummed with checksum in `sha256` header.

Accept-encoding **should** be added to headers especially if body is nonstandard (non-json)

Minmum size of body transport **must** transfer is 4MiB

### Discovery

Persistent nodes should send a message describing service every $heartbeat interval; it should be send to `discovery/node.$nodename` ; body should contain json-encoded:

* `node-name`
* `node-uuid`
* `node-info` - key-value map with node info
* `node-pubkey` - node's signing pubkey if any. Encoded in same way as signature with first byte being type, second being length, third being key, all encoded in base64
* `ts` -  RFC3399 or unixtime (plain or float for sub-ms accuracy if needed) timestamp
* `hb-interval` - configured interval between heartbeats. Max 120s.
* `ttl` - time after node should be considered inactive if it didn't send hb. should be 3x hb-interval
* `services` - hash of services node is providing and their relevant info; same as in discovery.service

Note that some attributes are same as in header; this is to facilitate for proxies and ability to cache state without re-serializing.


### Service info format

Service information should provide both its service message format and a brief documentation for what each option does. That allows for easily adding new APIs as having default + its documentation always available makes it easy to present it as something easy to interact with (like a web form with each field already filled with defaults and having corresponding docs in-place)

Service info sent to heartbeat and returned by service catalog is divided into two parts:

First part [key `fields_default`]  defines message structure together with default values for it (values that will be assumed by service if not provided by sender).

Second part [key `fields_description`]  defines a description for each field in human readable, markdown-formatted format:

* key just have its description as a value: `timeout: "timeout for a job; **0** specifies that job have no timeout`
* arrays will have its fields newline-joined before passing to markdown marker: `hosts: ["a list of hosts","that will be pinged"]`
* maps will use special key named `_comment_` as value for documentation, like that:

in addition of that there are 3 special fields in root, `_comment_top`, `_comment_body`, `_comment_bottom_`,  that are for adding header and footer to auto-generated documentation.
Top should contain short info about service and links to full documentation and project's root repos. It will be used as **summary** when needed (like displaying list of services with their abbreviated description)

Body should contain main part of documentation, ideally enough for basic service usage

Bottom will be appended to end of auto-generated docs


    db:
        _comment_: DB connection confi
        user: "db username"
        password: "db password"
        host: "db hostname; uses local socket if IP is not specified

Example:

    fields_default:
        cluster_hosts:
            - localhost
        db_conn:
            user: guest
            pass: guest
            host: localhost
        timeout: 60
    field_description:
        _comment_top_: DB API service: provides limited set of DB queries over API and caches the answers where applicable; project at http://example.com
        _comment_body_: **cluster_hosts** are other hosts in the cluster; that is used to ensure cache consistency when required. So far only MySQL is supported.
        _comment_bottom_: Licenced under LGPL v3
        cluster_hosts:
            - list of hosts that will be tried when looking for cluster; port is optional
            -
            - **note:** use [] for ipv6 addresses
        db_conn:
            _comment_: DB connection parameters. Only MySQL is supported so far, schema is in sql/schema.sql in main app dir
            user: username
            pass: password
            host: hostname, with optional port
