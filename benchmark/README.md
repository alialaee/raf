# RAF Benchmark Results


### APIResponse - Marshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| **RAF** | 1,096 (1.0x) | 2,008 | 2 |
| JSON | 1,576 (1.4x) | 1,460 | 2 |
| MsgPack | 1,905 (1.7x) | 3,113 | 12 |
| CBOR | 1,160 (1.1x) | 1,220 | 2 |
| BSON | 2,980 (2.7x) | 2,670 | 2 |

### APIResponse - Unmarshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| **RAF** | 1,461 (1.0x) | 1,707 | 57 |
| JSON | 8,886 (6.1x) | 3,167 | 69 |
| MsgPack | 3,280 (2.2x) | 2,354 | 56 |
| CBOR | 3,812 (2.6x) | 1,705 | 57 |
| BSON | 6,722 (4.6x) | 4,754 | 196 |

### Player - Marshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| RAF | 948 (1.0x) | 2,035 | 2 |
| JSON | 1,379 (1.5x) | 1,369 | 2 |
| MsgPack | 1,967 (2.1x) | 2,347 | 6 |
| CBOR | 1,192 (1.3x) | 1,054 | 2 |
| BSON | 2,951 (3.1x) | 1,407 | 2 |
| **Protobuf** | 887 (0.9x) | 340 | 1 |

### Player - Unmarshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| **RAF** | 980 (1.0x) | 927 | 25 |
| JSON | 8,230 (8.4x) | 1,763 | 36 |
| MsgPack | 3,206 (3.3x) | 1,314 | 28 |
| CBOR | 3,550 (3.6x) | 928 | 25 |
| BSON | 6,209 (6.3x) | 2,891 | 157 |
| Protobuf | 1,503 (1.5x) | 2,082 | 46 |

### TelemetryEvent - Marshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| **RAF** | 186 (1.0x) | 464 | 2 |
| JSON | 346 (1.9x) | 417 | 2 |
| MsgPack | 369 (2.0x) | 640 | 5 |
| CBOR | 231 (1.2x) | 360 | 2 |
| BSON | 478 (2.6x) | 427 | 2 |

### TelemetryEvent - Unmarshal

| Codec | ns/op | B/op | allocs/op |
|-------|------:|-----:|----------:|
| **RAF** | 202 (1.0x) | 271 | 9 |
| JSON | 1,678 (8.3x) | 530 | 15 |
| MsgPack | 540 (2.7x) | 319 | 10 |
| CBOR | 619 (3.1x) | 271 | 9 |
| BSON | 904 (4.5x) | 518 | 30 |
