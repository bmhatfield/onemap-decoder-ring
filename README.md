# Valheim Explored Decoder

Go decoder for Valheim `.explored` files written by `One Map To Rule Them All`
and `ServerSideMap`.

## Run

```sh
go run . fixtures/Dedicated.one_map_to_rule_them_all.explored
go run . fixtures/Dedicated.mod.serversidemap.explored
go run . --summary fixtures/Dedicated.one_map_to_rule_them_all.explored
go run . --pins fixtures/Dedicated.one_map_to_rule_them_all.explored
go run . --json decoded.json fixtures/Dedicated.one_map_to_rule_them_all.explored
```

## Supported Formats

| Format | Version | Map encoding | Pin owner field |
| --- | ---: | --- | --- |
| `one_map_to_rule_them_all` | 1 | bool bytes | yes |
| `one_map_to_rule_them_all` | 2 | packed bits | yes |
| `serversidemap` | 3 | bool bytes | no |

## Test

```sh
go test ./...
```

## Duplicate Pins Observed

The raw fixture data contains many repeated pins with the same name, type, and exact position. That matches the mod's `ArePinsEqual(SharedPin, SharedPin)` identity check.

| File | Raw pins | Duplicate groups | Extra duplicate pins | Unique by name/type/position |
| --- | ---: | ---: | ---: | ---: |
| `fixtures/Dedicated.one_map_to_rule_them_all.explored` | 325 | 41 | 258 | 67 |
| `fixtures/Dedicated.one_map_to_rule_them_all.explored.old` | 39,255 | 1,404 | 33,824 | 5,431 |

Largest examples:

| File | Repeated pin | Count |
| --- | --- | ---: |
| current | `Tu 178` at `(806.4286, 39.3086, -811.5291)` | 27 |
| old | `Tu 386` at `(804.3380, 39.9261, -805.8654)` | 3,794 |

Example excerpt from `--pins` showing the same repeated pin twice:

```json
[
  {
    "name": "Tu 178",
    "decoded_name": "Turnip",
    "pos": { "x": 806.4286, "y": 39.3086, "z": -811.5291 },
    "type": 3,
    "checked": true,
    "owner_id": "auto"
  },
  {
    "name": "Tu 178",
    "decoded_name": "Turnip",
    "pos": { "x": 806.4286, "y": 39.3086, "z": -811.5291 },
    "type": 3,
    "checked": true,
    "owner_id": "auto"
  }
]
```

The decoder reports raw data as stored; it does not deduplicate pins.
