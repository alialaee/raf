# RAF Benchmark Results


## APIResponse — Marshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 1,087 **fastest** | 2,009 | 2 |
| JSON | 1,558 (1.4x) | 1,460 | 2 |
| MsgPack | 1,974 (1.8x) | 3,114 | 12 |
| CBOR | 1,171 (1.1x) | 1,220 | 2 |
| BSON | 3,031 (2.8x) | 2,672 | 2 |

## APIResponse — Unmarshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 1,452 **fastest** | 1,707 | 57 |
| JSON | 8,894 (6.1x) | 3,167 | 69 |
| MsgPack | 3,258 (2.2x) | 2,355 | 56 |
| CBOR | 3,754 (2.6x) | 1,705 | 57 |
| BSON | 6,742 (4.6x) | 4,755 | 196 |

## Player — Marshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 1,028 **fastest** | 2,036 | 2 |
| JSON | 1,415 (1.4x) | 1,369 | 2 |
| MsgPack | 2,006 (2.0x) | 2,347 | 6 |
| CBOR | 1,199 (1.2x) | 1,054 | 2 |
| BSON | 2,912 (2.8x) | 1,408 | 2 |

## Player — Unmarshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 993 **fastest** | 927 | 25 |
| JSON | 8,288 (8.3x) | 1,763 | 36 |
| MsgPack | 3,238 (3.3x) | 1,314 | 28 |
| CBOR | 3,588 (3.6x) | 928 | 25 |
| BSON | 6,266 (6.3x) | 2,892 | 157 |

## TelemetryEvent — Marshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 190 **fastest** | 464 | 2 |
| JSON | 349 (1.8x) | 417 | 2 |
| MsgPack | 380 (2.0x) | 640 | 5 |
| CBOR | 239 (1.3x) | 360 | 2 |
| BSON | 478 (2.5x) | 427 | 2 |

## TelemetryEvent — Unmarshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 202 **fastest** | 271 | 9 |
| JSON | 1,660 (8.2x) | 530 | 15 |
| MsgPack | 539 (2.7x) | 320 | 10 |
| CBOR | 622 (3.1x) | 271 | 9 |
| BSON | 912 (4.5x) | 518 | 30 |
