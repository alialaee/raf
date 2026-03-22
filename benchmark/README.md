# RAF Benchmark Results


## APIResponse — Marshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 1,516 (1.3x) | 2,025 | 2 |
| JSON | 1,542 (1.3x) | 1,460 | 2 |
| MsgPack | 2,046 (1.8x) | 3,114 | 12 |
| CBOR | 1,162 **fastest** | 1,220 | 2 |
| BSON | 2,950 (2.5x) | 2,672 | 2 |

## APIResponse — Unmarshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 1,427 **fastest** | 1,707 | 57 |
| JSON | 8,773 (6.1x) | 3,167 | 69 |
| MsgPack | 3,248 (2.3x) | 2,354 | 56 |
| CBOR | 3,760 (2.6x) | 1,705 | 57 |
| BSON | 6,696 (4.7x) | 4,755 | 196 |

## Player — Marshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 1,402 (1.2x) | 2,040 | 2 |
| JSON | 1,394 (1.2x) | 1,369 | 2 |
| MsgPack | 2,020 (1.7x) | 2,347 | 6 |
| CBOR | 1,210 **fastest** | 1,054 | 2 |
| BSON | 2,942 (2.4x) | 1,409 | 2 |

## Player — Unmarshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 983 **fastest** | 927 | 25 |
| JSON | 8,298 (8.4x) | 1,763 | 36 |
| MsgPack | 3,182 (3.2x) | 1,314 | 28 |
| CBOR | 3,552 (3.6x) | 928 | 25 |
| BSON | 6,580 (6.7x) | 2,892 | 157 |

## TelemetryEvent — Marshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 255 (1.1x) | 464 | 2 |
| JSON | 359 (1.5x) | 417 | 2 |
| MsgPack | 378 (1.6x) | 640 | 5 |
| CBOR | 242 **fastest** | 360 | 2 |
| BSON | 486 (2.0x) | 427 | 2 |

## TelemetryEvent — Unmarshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 209 **fastest** | 271 | 9 |
| JSON | 1,694 (8.1x) | 530 | 15 |
| MsgPack | 559 (2.7x) | 320 | 10 |
| CBOR | 633 (3.0x) | 271 | 9 |
| BSON | 949 (4.5x) | 518 | 30 |
