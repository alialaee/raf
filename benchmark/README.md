# RAF Benchmark Results


## APIResponse — Marshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 1,193 (1.0x) | 2,009 | 2 |
| JSON | 1,590 (1.3x) | 1,460 | 2 |
| MsgPack | 1,972 (1.7x) | 3,114 | 12 |
| CBOR | 1,180 **fastest** | 1,220 | 2 |
| BSON | 3,040 (2.6x) | 2,672 | 2 |

## APIResponse — Unmarshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 1,454 **fastest** | 1,707 | 57 |
| JSON | 9,074 (6.2x) | 3,167 | 69 |
| MsgPack | 3,312 (2.3x) | 2,355 | 56 |
| CBOR | 3,776 (2.6x) | 1,705 | 57 |
| BSON | 6,786 (4.7x) | 4,755 | 196 |

## Player — Marshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 1,101 **fastest** | 2,036 | 2 |
| JSON | 1,401 (1.3x) | 1,369 | 2 |
| MsgPack | 2,005 (1.8x) | 2,347 | 6 |
| CBOR | 1,211 (1.1x) | 1,054 | 2 |
| BSON | 2,938 (2.7x) | 1,409 | 2 |

## Player — Unmarshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 990 **fastest** | 927 | 25 |
| JSON | 8,384 (8.5x) | 1,763 | 36 |
| MsgPack | 3,220 (3.3x) | 1,314 | 28 |
| CBOR | 3,521 (3.6x) | 928 | 25 |
| BSON | 6,264 (6.3x) | 2,892 | 157 |

## TelemetryEvent — Marshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 204 **fastest** | 464 | 2 |
| JSON | 347 (1.7x) | 417 | 2 |
| MsgPack | 382 (1.9x) | 640 | 5 |
| CBOR | 239 (1.2x) | 360 | 2 |
| BSON | 476 (2.3x) | 427 | 2 |

## TelemetryEvent — Unmarshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 199 **fastest** | 271 | 9 |
| JSON | 1,678 (8.4x) | 530 | 15 |
| MsgPack | 548 (2.8x) | 320 | 10 |
| CBOR | 606 (3.0x) | 271 | 9 |
| BSON | 913 (4.6x) | 518 | 30 |
