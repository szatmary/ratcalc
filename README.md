# ratcalc

A line-by-line calculator with exact rational arithmetic and compound unit support, built with [Gio](https://gioui.org).

Each line is an independent expression or assignment. Results appear in the right gutter as you type.

## Features

- **Exact rational arithmetic** — all math uses `math/big.Rat`, no floating-point rounding
- **Smart display** — fractions when denominator ≤ 1000 (`1/3`, `22/7`), decimals otherwise
- **Units** — length, weight, time, and volume with automatic conversion
- **Compound units** — `10 miles / gallon` → `10 mi/gal`, `5 m * 3 s` → `15 m*s`
- **No auto-cancellation** — `10 mi / 2 mi` → `5 mi/mi`, preserving the full dimensional trail
- **Bare unit words** — `gallon` without a number implies `1 gal`
- **Variables** — single or multi-word: `tax rate = 0.08`
- **File I/O** — open/save with Cmd+O / Cmd+S

## Building

Requires Go 1.25+.

```
go build -o ratcalc ./app
./ratcalc [file.txt]
```

## Examples

```
2 + 3                  → 5
1/3 + 1/6              → 1/2
5 meters + 100 cm      → 6 m
10 miles / gallon      → 10 mi/gal
100 mi / 5 gal         → 20 mi/gal
tax rate = 0.08
100 * tax rate          → 8
```

## Units

| Category | Units |
|----------|-------|
| Length   | mm, cm, m, km, in, ft, yd, mi |
| Weight   | mg, g, kg, oz, lb |
| Time     | ms, s, min, hr |
| Volume   | mL, L, floz, cup, pt, qt, gal |

Full names and plurals are also accepted (`meters`, `pounds`, `gallons`, etc.).

See [LANGUAGE.md](LANGUAGE.md) for the full language specification.
