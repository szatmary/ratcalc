# Ratcalc Language Specification

Ratcalc is a line-by-line calculator. Each line is an independent expression or
assignment, evaluated using exact rational arithmetic (`math/big.Rat`). Results
appear in the right gutter.

## Grammar

```
line        → assignment | conversion | expression | <empty>
assignment  → varname "=" ( conversion | expression )
conversion  → expression "to" ( compound_unit_spec | TIMEZONE | "unix" | "hex" | "bin" | "oct" )
compound_unit_spec → UNIT (("/" | "*") UNIT)*
expression  → term ( ("+" | "-") term )*
term        → unary ( ("*" | "/") unary )*
unary       → "-" unary | postfix
postfix     → primary ( unit | AMPM? TIMEZONE? )?
primary     → number | "@" DATESPEC | time | funccall | varname | "(" expression ")"
number      → NUMBER ( "." NUMBER )? ( "/" NUMBER )?
time        → TIME                            // HH:MM or HH:MM:SS
funccall    → WORD "(" [ expression ("," expression)* ] ")"
varname     → WORD ( WORD )*                  // greedy, multi-word
unit        → UNIT                            // matched from known units table
```

## Tokens

| Token     | Pattern                     |
|-----------|-----------------------------|
| `NUMBER`  | `[0-9]+` or `0x[0-9a-fA-F]+` or `0b[01]+` or `0o[0-7]+` |
| `WORD`    | `[a-zA-Z_][a-zA-Z0-9_]*`   |
| `PLUS`    | `+`                         |
| `MINUS`   | `-`                         |
| `STAR`    | `*`                         |
| `SLASH`   | `/`                         |
| `LPAREN`  | `(`                         |
| `RPAREN`  | `)`                         |
| `EQUALS`  | `=`                         |
| `DOT`     | `.`                         |
| `COMMA`   | `,`                         |
| `AT`      | `@` followed by date/time/number |
| `TIME`    | `H:MM` or `HH:MM[:SS]`      |
| `EOF`     |                             |

Whitespace is skipped between tokens.

`AT` tokens are recognized when `@` is followed by a date, datetime, time, or
plain digits (unix timestamp). Supported formats:

- `@YYYY-MM-DD` or `@YYYY-M-D` — date (1 or 2 digit month/day)
- `@YYYY-MM-DDTHH:MM:SS` — datetime with `T` separator
- `@YYYY-MM-DD HH:MM:SS` — datetime with space separator
- `@YYYY-MM-DD HH:MM:SS +NNNN` — datetime with UTC offset
- `@YYYY-MM-DD HH:MM:SS -NNNN` — datetime with UTC offset
- `@HH:MM` or `@HH:MM:SS` — time of day
- `@DIGITS` — unix timestamp

The `@` prefix is required for dates — `2024-01-31` without `@` is arithmetic
(`2024 - 1 - 31 = 1992`). Named timezone abbreviations (`PST`, `UTC`, etc.)
are handled as a separate postfix token: `@2024-01-31 10:30:00 PST`.

Time tokens are recognized when a 1-2 digit number is immediately followed by
`:MM` or `:MM:SS`. `12:00` is a time; `12 : 00` is not (the `:` is unknown and
skipped).

`AM` and `PM` (case-insensitive) are recognized as postfix modifiers on
time-producing expressions. They follow the standard 12-hour clock convention:
- `12:00 AM` = midnight (00:00), `12:30 AM` = 00:30
- `1:00 AM` through `11:59 AM` = unchanged (01:00–11:59)
- `12:00 PM` = noon (12:00), `12:30 PM` = 12:30
- `1:00 PM` through `11:59 PM` = add 12 hours (13:00–23:59)

AM/PM is consumed before timezone, so `3:30 PM PST` works as expected.

## Types

### Rational Numbers
All numbers are exact rationals (`math/big.Rat`). Literals:

- Integer: `42`, `-7`
- Hex: `0xFF`, `0x1A3` (case-insensitive prefix and digits)
- Binary: `0b1010`, `0b11110000`
- Octal: `0o77`, `0o755`
- Decimal: `3.14` (stored as `314/100`, auto-simplified)
- Fraction: `1/3`, `22/7`

### Time Values
A value may be a **time** (unix timestamp). When `IsTime` is true, the rational
number holds unix seconds and the unit is nil. Time is NOT a unit category —
it's a flag on the value to avoid compound-unit complications.

**Creating time values:**
- `Date(y, m, d)` — date at midnight UTC
- `Date(y, m, d, h, min, s)` — date with time, UTC
- `Time(h, m)` or `Time(h, m, s)` — time-of-day today, UTC
- `@2024-01-31` — sugar for `Date(2024, 1, 31)`
- `@2024-01-31T10:30:00` — sugar for `Date(2024, 1, 31, 10, 30, 0)`
- `@2024-01-31 10:30:00` — same (space instead of `T`)
- `@2024-01-31 10:30:00 +0530` — datetime with UTC offset
- `@2024-01-31 10:30:00 PST` — datetime with named timezone (postfix)
- `@14:30` — sugar for `Time(14, 30)`
- `@14:30:00` — sugar for `Time(14, 30, 0)`
- `@1706745600` — sugar for `Unix(1706745600)`
- Time-of-day literal: `12:00` or `14:30:00` (today's date in UTC)
- `Unix(n)` — interprets number as unix timestamp, auto-detects precision:
  - `< 1e12` → seconds
  - `< 1e15` → milliseconds (÷1000)
  - `< 1e18` → microseconds (÷1e6)
  - `≥ 1e18` → nanoseconds (÷1e9)
- `Now()` — returns current time, updates every second

**Timezones:**

A timezone abbreviation can follow a time-producing expression as a postfix
(`12:00 PST`) to indicate the input timezone, or appear after `to` to convert
display (`Now() to EST`).

Postfix (input timezone): `12:00 PST` means noon in PST. Internally the value
is adjusted to UTC (noon PST = 20:00 UTC). The display timezone is set to PST.

Conversion (`to TZ`): `Now() to EST` displays the current time in EST. The
internal UTC value is unchanged; only the display timezone changes.

Supported timezone abbreviations:

| Abbrev | UTC Offset |
|--------|------------|
| UTC    | +00:00     |
| GMT    | +00:00     |
| EST    | -05:00     |
| EDT    | -04:00     |
| CST    | -06:00     |
| CDT    | -05:00     |
| MST    | -07:00     |
| MDT    | -06:00     |
| PST    | -08:00     |
| PDT    | -07:00     |
| CET    | +01:00     |
| CEST   | +02:00     |
| IST    | +05:30     |
| JST    | +09:00     |
| AEST   | +10:00     |
| AEDT   | +11:00     |
| NZST   | +12:00     |
| NZDT   | +13:00     |

Timezone names are context-sensitive: `PST` is only treated as a timezone when
it follows a time-producing node (postfix) or appears after `to`. Otherwise it
could be a variable name.

### Duration Values
A **duration** is a value with a time-category unit (`s`, `min`, `hr`, `d`, `wk`,
`yr`, `ms`). Duration is not a separate type — it reuses the existing unit system.
Durations can be converted between time units with `to`:

```
86400 s to hr      → 24 hr
24 hr to d         → 1 d
1 wk to d          → 7 d
```

**Time arithmetic:**
- `time ± duration` → time (convert duration to seconds, add/subtract)
- `time ± number` → **error** ("use a time unit like s, hr, d")
- `time - time` → duration (result in seconds: `86400 s`)
- `time + time` → error
- `time * anything` → error
- `time / anything` → error

Duration results from `time - time` can be converted to other time units:

```
@2024-02-01 - @2024-01-31           → 86400 s
@2024-02-01 - @2024-01-31 to hr     → 24 hr
@2024-02-01 - @2024-01-31 to d      → 1 d
```

Adding durations to times:

```
@2024-01-31 + 86400 s   → 2024-02-01 00:00:00 +0000
@2024-01-31 + 24 hr     → 2024-02-01 00:00:00 +0000
@2024-01-31 + 1 d       → 2024-02-01 00:00:00 +0000
@2024-02-01 - 1 hr      → 2024-01-31 23:00:00 +0000
```

**Display:** UTC format `2024-01-31 10:30:00 +0000`, or with timezone
`2024-01-31 04:30:00 -0800` when a timezone is set.

### Values with Units
A value may carry an optional unit. Arithmetic checks unit compatibility:

- Same-category units are converted to a common base before operating
- Multiplying/dividing values with units produces compound results or strips units
- Adding incompatible units is an error

## Variables

Variable names may contain spaces. They are sequences of WORD tokens that do
not match a known unit when following a number.

```
x = 10
my variable = 42
my variable * 2        → 84
```

Assignment uses `=`. When a line contains multiple `=`, the **last** one splits
the variable name from the expression.

## Functions

| Function | Args | Description |
|----------|------|-------------|
| `Now()`  | 0    | Current UTC time, updates every second |
| `Date(y, m, d)` | 3 | Date at midnight UTC |
| `Date(y, m, d, h, min, s)` | 6 | Date with time, UTC |
| `Time(h, m)` | 2 | Time-of-day today, UTC (seconds = 0) |
| `Time(h, m, s)` | 3 | Time-of-day today, UTC |
| `Unix(n)` | 1 | Unix timestamp (auto-detects s/ms/μs/ns) |

## Units

### Length
| Short | Full       | Base (meters) |
|-------|------------|---------------|
| mm    | millimeters| 0.001         |
| cm    | centimeters| 0.01          |
| m     | meters     | 1             |
| km    | kilometers | 1000          |
| in    | inches     | 0.0254        |
| ft    | feet       | 0.3048        |
| yd    | yards      | 0.9144        |
| mi    | miles      | 1609.344      |

### Weight
| Short | Full       | Base (grams)  |
|-------|------------|---------------|
| mg    | milligrams | 0.001         |
| g     | grams      | 1             |
| kg    | kilograms  | 1000          |
| oz    | ounces     | 28.3495       |
| lb    | pounds     | 453.592       |

### Time
| Short | Full       | Base (seconds)|
|-------|------------|---------------|
| ms    | milliseconds| 0.001        |
| s     | seconds    | 1             |
| min   | minutes    | 60            |
| hr    | hours      | 3600          |
| d     | days       | 86400         |
| wk    | weeks      | 604800        |
| yr    | years      | 31557600 (365.25 days) |

### Volume
| Short | Full       | Base (mL)     |
|-------|------------|---------------|
| mL    | milliliters| 1             |
| L     | liters     | 1000          |
| floz  | floz       | 29.5735       |
| cup   | cups       | 236.588       |
| pt    | pints      | 473.176       |
| qt    | quarts     | 946.353       |
| gal   | gallons    | 3785.41       |

Both suffix (`5m`) and full name (`5 meters`) forms are supported.

## Compound Units

Arithmetic on values with units produces compound units. Units are never
auto-cancelled — the full dimensional trail is preserved.

- **Division** produces ratio units: `10 miles / 1 gallon` → `10 mi/gal`
- **Multiplication** produces product units: `5 m * 3 s` → `15 m*s`
- **Same-category division**: `10 mi / 2 mi` → `5 mi/mi` (no cancellation)

### Bare unit words

A bare unit name (like `gallon`) without a preceding number is treated as `1`
of that unit. This means `10 miles / gallon` works naturally:

```
10 miles / gallon     → 10 mi/gal
```

### Add/Sub with compound units

Adding or subtracting compound units requires compatible units (same categories
at each position). Units are converted to the left operand's units:

```
10 mi/gal + 5 mi/gal  → 15 mi/gal
```

## Unit Conversion with `to`

The `to` keyword converts a value to a target unit or compound unit. It has the
lowest precedence, so the entire expression is evaluated before conversion:

```
100 km to mi              → converts 100 km to miles
5 m + 300 cm to km        → evaluates sum (8 m), then converts to km
40 mi / 1 gal to km/L     → evaluates compound (40 mi/gal), then converts to km/L
100 km/hr to mi/hr         → speed conversion
```

The target unit spec supports compound units with `*` and `/`:

```
compound_unit_spec → UNIT (("/" | "*") UNIT)*
```

Conversion requires compatible dimensions — converting between incompatible
units (e.g. `5 m to kg`) is an error.

### `to unix`

`to unix` converts a time value back to its raw unix timestamp (seconds since
epoch). Fractional seconds are preserved as exact rationals:

```
@2024-02-01 to unix           → 1706745600
(@2024-02-01 + 1/2 s) to unix → 1706745600.5
Now() to unix                  → current unix timestamp
```

### `to hex`, `to bin`, `to oct`

`to hex`, `to bin`, and `to oct` convert an integer value to hexadecimal, binary,
or octal display format. The value must be an integer — non-integer values produce
an error.

```
255 to hex        → 0xff
10 to bin         → 0b1010
63 to oct         → 0o77
0xFF to hex       → 0xff   (round-trip)
0xFF + 1 to hex   → 0x100
```

`to` is a **context-sensitive keyword**: it is only treated as the conversion
operator when immediately followed by a known unit name or timezone abbreviation.
Otherwise `to` is a valid variable name.

## Operators

| Op  | Precedence | Associativity |
|-----|------------|---------------|
| `+` | 1          | Left          |
| `-` | 1          | Left          |
| `*` | 2          | Left          |
| `/` | 2          | Left          |
| `-` (unary) | 3  | Right        |

Parentheses override precedence.

## Display

Results use smart formatting:
- If the denominator is ≤ 1000 and the fraction doesn't simplify to an integer,
  display as a fraction: `1/3`, `22/7`
- Otherwise display as a decimal (truncated to reasonable precision)
- Values with units append the unit string: `5 m`, `2.5 kg`, `20 mi/gal`
- Compound units display as `num/den` with `*` separating multiple units on a side: `m*kg/s*s`

## Examples

```
2 + 3                  → 5
1/3 + 1/6              → 1/2
x = 10                 → 10
x + 5                  → 15
my variable = 42       → 42
my variable * 2        → 84
5 meters + 100 cm      → 6 m
10 miles / gallon      → 10 mi/gal
100 mi / 5 gal         → 20 mi/gal
10 mi / 2 mi           → 5 mi/mi
5 m * 3 s              → 15 m*s
@2024-01-31            → 2024-01-31 00:00:00 +0000
@2024-01-31T10:30:00   → 2024-01-31 10:30:00 +0000
Date(2024, 1, 31)      → 2024-01-31 00:00:00 +0000
2024-01-31             → 1992  (arithmetic, no @ prefix)
2024 - 01 - 31         → 1992  (arithmetic)
Unix(1706745600)       → 2024-02-01 00:00:00 +0000
@1706745600            → 2024-02-01 00:00:00 +0000
Unix(1706745600000)    → 2024-02-01 00:00:00 +0000  (auto-detect ms)
14:30                  → today 14:30:00 +0000 (time-of-day literal)
@14:30                 → today 14:30:00 +0000 (@ time sugar)
Time(14, 30)           → today 14:30:00 +0000
3:30 PM                → today 15:30:00 +0000 (AM/PM)
12:00 AM               → today 00:00:00 +0000 (midnight)
12:00 PM               → today 12:00:00 +0000 (noon)
3:30 PM PST            → today 15:30:00 -0800 (AM/PM + timezone)
12:00 PST              → today 12:00:00 -0800 (input timezone)
12:00 PST to UTC       → today 20:00:00 +0000 (round-trip)
12:00 UTC to PST       → today 04:00:00 -0800 (timezone conversion)
Now()                  → current UTC time, updates every second
Now() to EST           → current time in EST
Now() - @2024-01-01    → duration in seconds since Jan 1 2024
@2024-01-31 + 1 d      → 2024-02-01 00:00:00 +0000
@2024-01-31 + 24 hr    → 2024-02-01 00:00:00 +0000
@2024-01-31 + 86400 s  → 2024-02-01 00:00:00 +0000
@2024-02-01 - @2024-01-31         → 86400 s
@2024-02-01 - @2024-01-31 to hr   → 24 hr
@2024-02-01 - @2024-01-31 to d    → 1 d
@2024-01-31 10:30:00 PST   → 2024-01-31 10:30:00 -0800
@2024-01-31 02:30:00 -0800 → 2024-01-31 10:30:00 +0000 (offset round-trip)
@2024-01-31T10:30:00 to PST → 2024-01-31 02:30:00 -0800
@2024-02-01 to unix    → 1706745600
Now() to unix          → current unix timestamp
0xFF                   → 255
0b1010                 → 10
0o77                   → 63
255 to hex             → 0xff
10 to bin              → 0b1010
63 to oct              → 0o77
0xFF + 1               → 256
100 km to mi           → ~62.14 mi
40 mi / 1 gal to km/L  → ~17.01 km/L
5 m + 300 cm to km     → 0.008 km
```
