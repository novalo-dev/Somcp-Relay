# Relay protocol

How O'So AI Relay (`oso`) and the somcp module of a PrestaShop shop talk. The relay
always calls the shop; the shop never calls the relay, so nothing has to be
opened on the merchant's network.

## Address

Every call goes to the module's front controller, which works with friendly
URLs on or off:

```
<shop>/index.php?fc=module&module=somcp&controller=relay&action=<action>
```

The shop address must be `https://`. Plain `http://` is accepted only for
`localhost`, `127.0.0.1`, private `192.168.*` and `10.*` addresses and
`*.local` names, where a test shop runs on the same network.

Bodies are JSON. Every action but `pair` carries the relay's token:
`Authorization: Bearer <token>`. A `401` on any of them means the token was
revoked: the relay stops and asks to be linked again.

## pair - trade a pairing code for a token

`POST action=pair`

```json
{"code": "K7Q2-9XD4", "relay": {"version": "0.1.0", "os": "windows/amd64"}}
```

The code is the one the back office shows under **Built-in assistant > Add a
local assistant**: single use, valid 10 minutes, upper case. Answer:

```json
{"token": "<secret>", "shop": {"name": "MyShop", "version": "9.0.0"}}
```

A wrong or expired code: `400 {"error": "..."}`. The shop keeps only a hash
of the token.

## hello - say which AI answers

`POST action=hello`, sent when the relay starts:

```json
{"ai": {"software": "Ollama", "models": [{"id": "qwen2.5:7b", "tools": true}, {"id": "gemma2:2b", "tools": false}]}}
```

`tools` is the result of the relay's tool test; it is absent for a model not
tested. The back office offers only the models with `"tools": true`. Answer:
`{"shop": {"name": "MyShop"}}`.

## next - wait for a chat request

`GET action=next`. The shop holds the request up to about 25 seconds while it
has nothing to send, then answers `204`, and the relay asks again. With a
request:

```json
{"job": {"id": "<id>", "request": {"model": "qwen2.5:7b", "messages": [...], "tools": [...]}}}
```

`request` is an OpenAI-compatible chat completion body. The relay refuses it
unless `model` is one it reported with `"tools": true` and `messages` is a
list; it forces `"stream": false`. The address of the AI is never taken from
the request: the relay only calls the address typed on the computer.

## result - send the answer back

`POST action=result`

```json
{"id": "<id>", "status": 200, "response": {"choices": [...], "usage": {...}}}
```

`status` and `response` are the AI's own, as they came. When there is no
answer - the request was refused, the AI did not respond - `error` replaces
`response`:

```json
{"id": "<id>", "status": 502, "error": "connection refused"}
```

Answer: `204`.
