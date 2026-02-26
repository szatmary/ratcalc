# Ratcalc Language Specification

Ratcalc is a line-by-line calculator. Each line is an independent expression or
assignment, evaluated using exact rational arithmetic. Results appear in the
right gutter.

## Grammar

```
line        → assignment | conversion | bitwise_or | <empty>
assignment  → varname "=" ( conversion | bitwise_or )
conversion  → bitwise_or "to" ( compound_unit_spec | TIMEZONE | "unix" | "hex" | "bin" | "oct" | "hms" )
compound_unit_spec → UNIT ("/" UNIT)?
bitwise_or  → bitwise_xor ( "|" bitwise_xor )*
bitwise_xor → bitwise_and ( "^" bitwise_and )*
bitwise_and → shift ( "&" shift )*
shift       → expression ( ("<<" | ">>") expression )*
expression  → term ( ("+" | "-") term )*
term        → unary ( ("*" | "/") unary )*
unary       → ("-" | "~") unary | exponent
exponent    → postfix ( "**" unary )?
postfix     → primary ( "!" | "%" | unit | AMPM? TIMEZONE? )?
primary     → number | "@" DATESPEC | time | funccall | varname | "#" NUMBER | CURRENCY primary | "(" bitwise_or ")"
number      → NUMBER ( "." NUMBER )? ( "/" NUMBER )?
time        → TIME                            // HH:MM or HH:MM:SS
funccall    → WORD "(" [ bitwise_or ("," bitwise_or)* ] ")"
varname     → WORD                            // single word, starts with letter
unit        → UNIT                            // matched from known units table
```

## Tokens

| Token      | Pattern                     |
|------------|-----------------------------|
| `NUMBER`   | `[0-9]+` or `0x[0-9a-fA-F]+` or `0b[01]+` or `0o[0-7]+` |
| `WORD`     | `[a-zA-Z_][a-zA-Z0-9_]*`   |
| `PLUS`     | `+`                         |
| `MINUS`    | `-`                         |
| `STAR`     | `*`                         |
| `BANG`     | `!`                         |
| `STARSTAR` | `**`                        |
| `SLASH`    | `/`                         |
| `AMP`      | `&`                         |
| `PIPE`     | `\|`                        |
| `CARET`    | `^`                         |
| `TILDE`    | `~`                         |
| `LSHIFT`   | `<<`                        |
| `RSHIFT`   | `>>`                        |
| `LPAREN`   | `(`                         |
| `RPAREN`   | `)`                         |
| `EQUALS`   | `=`                         |
| `DOT`      | `.`                         |
| `COMMA`    | `,`                         |
| `PERCENT`  | `%`                         |
| `HASH`     | `#`                         |
| `AT`       | `@` followed by date/time/number |
| `CURRENCY` | `$`, `€`, `£`, `¥`           |
| `TIME`     | `H:MM` or `HH:MM[:SS]`      |
| `EOF`      |                             |

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
All numbers are exact rationals. Literals:

- Integer: `42`, `-7`
- Hex: `0xFF`, `0x1A3` (case-insensitive prefix and digits)
- Binary: `0b1010`, `0b11110000`
- Octal: `0o77`, `0o755`
- Decimal: `3.14` (stored as `314/100`, auto-simplified)
- Fraction: `1/3`, `22/7`
- Percentage: `50%` = `1/2`, `10%` = `1/10` (divides by 100)

### Percentage

The `%` suffix divides a value by 100. It binds tighter than arithmetic operators,
so `200 * 10%` evaluates as `200 * 0.1 = 20`.

```
50%            → 1/2
10%            → 1/10
200 * 10%      → 20
rate = 5%      → 0.05
1000 * rate    → 50
```

## Variables

Variable names are single words that must start with a letter. They may contain
letters, digits, and underscores.

```
x = 10
price = 42
price * 2              → 84
```

Assignment uses `=`. The variable name is the single word before the first `=`.

### Line References

`#N` refers to the result of line N (1-indexed). Line references update
automatically when lines are inserted or removed.

```
100                    → 100
#1 * 2                 → 200
#1 + #2                → 300
```

## Units

Both suffix (`5m`) and full name (`5 meters`) forms are supported. A bare unit
name (like `gallon`) without a preceding number is treated as `1` of that unit.

### Length
| Short | Full       | Base (meters) |
|-------|------------|---------------|
| pm    | picometers | 1e-12         |
| nm    | nanometers | 1e-9          |
| um    | micrometers| 1e-6          |
| mm    | millimeters| 0.001         |
| cm    | centimeters| 0.01          |
| m     | meters     | 1             |
| km    | kilometers | 1000          |
| in    | inches     | 0.0254        |
| ft    | feet       | 0.3048        |
| yd    | yards      | 0.9144        |
| mi    | miles      | 1609.344      |
| au    | au         | 149597870700  |

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
| Short | Full       | Base (L)      |
|-------|------------|---------------|
| mL    | milliliters| 0.001         |
| L     | liters     | 1             |
| floz  | floz       | 0.0295735     |
| cup   | cups       | 0.236588      |
| pt    | pints      | 0.473176      |
| qt    | quarts     | 0.946353      |
| gal   | gallons    | 3.78541       |

### Temperature
Temperature conversion is offset-based, not purely multiplicative.
Temperature units cannot appear in compound units (no `C/s`).

| Short | Full        | Base (kelvin)                      |
|-------|-------------|------------------------------------|
| K     | kelvin      | 1 (no offset)                      |
| C     | celsius     | 1 (offset +273.15)                 |
| F     | fahrenheit  | 5/9 (offset +459.67)               |

```
100 C to F         → 212 F
0 C to K           → 273.15 K
32 F to C          → 0 C
0 K to C           → -273.15 C
```

### Pressure
| Short | Full        | Base (pascals) |
|-------|-------------|----------------|
| Pa    | pascals     | 1              |
| kPa   | kilopascals | 1000           |
| bar   | bars        | 100000         |
| atm   | atmospheres | 101325         |
| psi   | psi         | ~6894.757      |

### Force
| Short | Full        | Base (newtons) |
|-------|-------------|----------------|
| N     | newtons     | 1              |
| kN    | kilonewtons | 1000           |
| lbf   | lbf         | ~4.44822       |

### Energy
| Short | Full           | Base (joules) |
|-------|----------------|---------------|
| J     | joules         | 1             |
| kJ    | kilojoules     | 1000          |
| Wh    | watt-hours     | 3600          |
| kWh   | kilowatt-hours | 3600000       |
| cal   | calories       | 4.184         |
| kcal  | kilocalories   | 4184          |
| BTU   | BTU            | ~1055.06      |

### Power
| Short | Full        | Base (watts) |
|-------|-------------|--------------|
| W     | watts       | 1            |
| kW    | kilowatts   | 1000         |
| MW    | megawatts   | 1000000      |
| hp    | horsepower  | ~745.7       |

### Voltage
| Short | Full        | Base (volts) |
|-------|-------------|--------------|
| mV    | millivolts  | 0.001        |
| V     | volts       | 1            |
| kV    | kilovolts   | 1000         |

### Current
| Short | Full         | Base (amperes) |
|-------|--------------|----------------|
| mA    | milliamperes | 0.001          |
| A     | amperes      | 1              |

### Resistance
| Short | Full    | Base (ohms) |
|-------|---------|-------------|
| ohm   | ohms    | 1           |
| kohm  | kilohms | 1000        |

### Data
| Short | Full       | Base (bytes) |
|-------|------------|--------------|
| bit   | bits       | 1/8          |
| kbit  | kilobits   | 125          |
| Mbit  | megabits   | 125000       |
| Gbit  | gigabits   | 125000000    |
| Tbit  | terabits   | 125000000000 |
| Kibit | kibibits   | 128          |
| Mibit | mebibits   | 131072       |
| Gibit | gibibits   | 134217728    |
| Tibit | tebibits   | 137438953472 |
| B     | bytes      | 1            |
| KB    | kilobytes  | 1000         |
| MB    | megabytes  | 1e6          |
| GB    | gigabytes  | 1e9          |
| TB    | terabytes  | 1e12         |
| KiB   | kibibytes  | 1024         |
| MiB   | mebibytes  | 1048576      |
| GiB   | gibibytes  | 1073741824   |
| TiB   | tebibytes  | 1099511627776|

### Currency

Currency values are displayed with 2 decimal places. Currencies with known
symbols use prefix notation (`$80.00`, `€50.00`); others use suffix (`80.00 CAD`).
Currency symbols (`$`, `€`, `£`, `¥`) can be used as prefix operators.

All currencies are independent — there are no exchange rate conversions.

| Short | Symbol | Full    |
|-------|--------|---------|
| USD   | $      | dollars |
| EUR   | €      | euros   |
| GBP   | £      |         |
| JPY   | ¥      | yen     |
| CAD   |        |         |
| AUD   |        |         |
| CHF   |        |         |

```
$50 + $30          → $80.00
$100 * 1.08        → $108.00
€50                → €50.00
£75.50             → £75.50
¥1000              → ¥1000.00
50 USD             → $50.00
50 EUR             → €50.00
50 CAD             → 50.00 CAD  (no symbol, suffix)
$(50 + 30)         → $80.00
$4 / 1 hr          → $4.00/hr  (compound currency unit)
$240 / 1 hr to $/min → $4.00/min
```

## Compound Units

Arithmetic on values with units produces compound units. Each side (numerator
and denominator) holds at most one unit category. Categories cancel across
numerator/denominator during multiplication and division.

- **Division** produces ratio units: `10 miles / 1 gallon` → `10 mi/gal`
- **Multiplication with cancellation**: `60 mi/hr * 2 hr` → `120 mi` (hr cancels)
- **Same-category division**: `10 mi / 2 mi` → `5` (mi cancels)
- **Error**: operations producing >1 category per side (e.g. `5 m * 3 kg`) are errors

```
10 miles / gallon     → 10 mi/gal
60 mi/hr * 2 hr       → 120 mi  (time category cancels)
```

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

The target unit spec supports compound units with `/`:

```
compound_unit_spec → UNIT ("/" UNIT)?
```

Conversion requires compatible dimensions — converting between incompatible
units (e.g. `5 m to kg`) is an error.

### `to unix`

`to unix` converts a time value back to its raw unix timestamp (seconds since
epoch). Fractional seconds are preserved as exact rationals:

```
@2024-02-01 to unix           → 1706745600
(@2024-02-01 + 1/2 s) to unix → 1706745600.5
now() to unix                  → current unix timestamp
```

### `to hex`, `to bin`, `to oct`

`to hex`, `to bin`, and `to oct` convert an integer value to hexadecimal, binary,
or octal display format. The value must be an integer — non-integer values produce
an error. Physical units are stripped — only the numeric value is displayed.

```
255 to hex        → 0xff
10 to bin         → 0b1010
63 to oct         → 0o77
0xFF to hex       → 0xff   (round-trip)
0xFF + 1 to hex   → 0x100
255 B to hex      → 0xff   (units stripped)
```

### `to hms`

`to hms` formats a time or dimensionless value (in seconds) as hours, minutes,
and seconds:

```
3661 to hms       → 1h 1m 1s
2.5 hr to hms     → 2h 30m 0s
90 s to hms       → 1m 30s
```

`to` is a **context-sensitive keyword**: it is only treated as the conversion
operator when immediately followed by a known unit name or timezone abbreviation.
Otherwise `to` is a valid variable name.

## Time Values

A value may be a **time** (unix timestamp internally stored as seconds). Time is
a distinct value type — not a unit category.

**Creating time values:**
- `date(y, m, d)` — date at midnight UTC
- `date(y, m, d, h, min, s)` — date with time, UTC
- `time(h, m)` or `time(h, m, s)` — time-of-day today, UTC
- `@2024-01-31` — sugar for `date(2024, 1, 31)`
- `@2024-01-31T10:30:00` — sugar for `date(2024, 1, 31, 10, 30, 0)`
- `@2024-01-31 10:30:00` — same (space instead of `T`)
- `@2024-01-31 10:30:00 +0530` — datetime with UTC offset
- `@2024-01-31 10:30:00 PST` — datetime with named timezone (postfix)
- `@14:30` — sugar for `time(14, 30)`
- `@14:30:00` — sugar for `time(14, 30, 0)`
- `@1706745600` — sugar for `unix(1706745600)`
- Time-of-day literal: `12:00` or `14:30:00` (today's date in UTC)
- `unix(n)` — interprets number as unix timestamp, auto-detects precision:
  - `< 1e12` → seconds
  - `< 1e15` → milliseconds (÷1000)
  - `< 1e18` → microseconds (÷1e6)
  - `≥ 1e18` → nanoseconds (÷1e9)
- `now()` — returns current time, updates every second

**Timezones:**

A timezone abbreviation can follow a time-producing expression as a postfix
(`12:00 PST`) to indicate the input timezone, or appear after `to` to convert
display (`now() to EST`).

Postfix (input timezone): `12:00 PST` means noon in PST. Internally the value
is adjusted to UTC (noon PST = 20:00 UTC). The display timezone is set to PST.

Conversion (`to TZ`): `now() to EST` displays the current time in EST. The
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

## Functions

### Time Functions

| Function | Args | Description |
|----------|------|-------------|
| `now()`  | 0    | Current UTC time, updates every second |
| `date(y, m, d)` | 3 | Date at midnight UTC |
| `date(y, m, d, h, min, s)` | 6 | Date with time, UTC |
| `time(h, m)` | 2 | Time-of-day today, UTC (seconds = 0) |
| `time(h, m, s)` | 3 | Time-of-day today, UTC |
| `unix(n)` | 1 | Unix timestamp (auto-detects s/ms/μs/ns) |

### Time Extraction Functions

Extract components from a time value. Returns an integer.

| Function | Args | Description |
|----------|------|-------------|
| `year(t)` | 1 | Year (e.g. 2024) |
| `month(t)` | 1 | Month (1–12) |
| `day(t)` | 1 | Day of month (1–31) |
| `hour(t)` | 1 | Hour (0–23) |
| `minute(t)` | 1 | Minute (0–59) |
| `second(t)` | 1 | Second (0–59) |

### Math Functions

All math functions convert to float64 internally. Results are approximate.
Values with units or time flags are rejected.

| Function | Args | Description |
|----------|------|-------------|
| `sin(x)` | 1 | Sine (radians) |
| `cos(x)` | 1 | Cosine (radians) |
| `tan(x)` | 1 | Tangent (radians) |
| `asin(x)` | 1 | Arcsine (radians) |
| `acos(x)` | 1 | Arccosine (radians) |
| `atan(x)` | 1 | Arctangent (radians) |
| `sqrt(x)` | 1 | Square root |
| `abs(x)` | 1 | Absolute value |
| `log(x)` | 1 | Base-10 logarithm |
| `ln(x)` | 1 | Natural logarithm |
| `log2(x)` | 1 | Base-2 logarithm |
| `ceil(x)` | 1 | Ceiling (round up) |
| `floor(x)` | 1 | Floor (round down) |
| `round(x)` | 1 | Banker's rounding (round half to even) |
| `pow(x, y)` | 2 | x raised to the power y |
| `mod(x, y)` | 2 | Remainder of x / y |
| `min(x, y)` | 2 | Minimum of x and y |
| `max(x, y)` | 2 | Maximum of x and y |
| `atan2(y, x)` | 2 | Two-argument arctangent (radians) |

### Utility Functions

| Function | Args | Description |
|----------|------|-------------|
| `num(x)` | 1 | Strip units, return the display value as a pure number |

### Financial Functions

Financial functions use float64 math internally. All arguments must be
dimensionless.

| Function | Args | Description |
|----------|------|-------------|
| `fv(rate, n, pmt)` | 3 | Future value of annuity: `pmt * ((1+rate)^n - 1) / rate` |
| `pv(rate, n, pmt)` | 3 | Present value of annuity: `pmt * (1 - (1+rate)^(-n)) / rate` |

```
fv(0.05, 10, 1000)   → ~12577.89  (future value at 5% for 10 periods)
pv(0.05, 10, 1000)   → ~7721.73   (present value at 5% for 10 periods)
```

### Constants

| Name | Value | Description |
|------|-------|-------------|
| `pi` | 3.141592653589793 | Ratio of circumference to diameter |
| `e`  | 2.718281828459045 | Euler's number |
| `c`  | 299792458 m/s | Speed of light |

## Operators

| Op  | Precedence | Associativity | Notes |
|-----|------------|---------------|-------|
| `\|`  | 1        | Left          | Bitwise OR (integers only) |
| `^`   | 2        | Left          | Bitwise XOR (integers only) |
| `&`   | 3        | Left          | Bitwise AND (integers only) |
| `<<`  | 4        | Left          | Left shift (integers only) |
| `>>`  | 4        | Left          | Right shift (integers only) |
| `+`   | 5        | Left          | Addition |
| `-`   | 5        | Left          | Subtraction |
| `*`   | 6        | Left          | Multiplication |
| `/`   | 6        | Left          | Division |
| `-` (unary) | 7  | Right         | Negation |
| `~`   | 7        | Right         | Bitwise NOT (integers only) |
| `**`  | 8        | Right         | Exponentiation |
| `!`   | 9        | Postfix       | Factorial (non-negative integers only) |

Parentheses override precedence.

Bitwise operations (`&`, `|`, `^`, `~`, `<<`, `>>`) require integer operands.
`**` uses exact rational arithmetic for integer exponents, float for non-integer.
`!` computes factorial using exact integer arithmetic (e.g. `20!` = `2432902008176640000`).

## Comments

Lines beginning with `;` or `//` (after optional whitespace) are comments and
produce no output.

## Display

Results use smart formatting:
- Fractions are shown if the result fits in the gutter width (e.g. `1/3`, `500/1001`)
- Otherwise displayed as decimal (truncated to 10 significant digits)
- Very large or very small values use scientific notation (e.g. `1.23e+15`)
- Values with units append the unit string: `5 m`, `2.5 kg`, `20 mi/gal`
- Compound units display as `num/den`: `mi/hr`, `km/L`

## Examples

```
2 + 3                  → 5
1/3 + 1/6              → 1/2
x = 10                 → 10
x + 5                  → 15
price = 42             → 42
price * 2              → 84
#1 + #2                → result of line 1 + result of line 2
5 meters + 100 cm      → 6 m
10 miles / gallon      → 10 mi/gal
100 mi / 5 gal         → 20 mi/gal
10 mi / 2 mi           → 5  (same category cancels)
60 mi/hr * 2 hr        → 120 mi  (time cancels)
@2024-01-31            → 2024-01-31 00:00:00 +0000
@2024-01-31T10:30:00   → 2024-01-31 10:30:00 +0000
date(2024, 1, 31)      → 2024-01-31 00:00:00 +0000
2024-01-31             → 1992  (arithmetic, no @ prefix)
2024 - 01 - 31         → 1992  (arithmetic)
unix(1706745600)       → 2024-02-01 00:00:00 +0000
@1706745600            → 2024-02-01 00:00:00 +0000
unix(1706745600000)    → 2024-02-01 00:00:00 +0000  (auto-detect ms)
14:30                  → today 14:30:00 +0000 (time-of-day literal)
@14:30                 → today 14:30:00 +0000 (@ time sugar)
time(14, 30)           → today 14:30:00 +0000
3:30 PM                → today 15:30:00 +0000 (AM/PM)
12:00 AM               → today 00:00:00 +0000 (midnight)
12:00 PM               → today 12:00:00 +0000 (noon)
3:30 PM PST            → today 15:30:00 -0800 (AM/PM + timezone)
12:00 PST              → today 12:00:00 -0800 (input timezone)
12:00 PST to UTC       → today 20:00:00 +0000 (round-trip)
12:00 UTC to PST       → today 04:00:00 -0800 (timezone conversion)
now()                  → current UTC time, updates every second
now() to EST           → current time in EST
now() - @2024-01-01    → duration in seconds since Jan 1 2024
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
now() to unix          → current unix timestamp
0xFF                   → 255
0b1010                 → 10
0o77                   → 63
255 to hex             → 0xff
10 to bin              → 0b1010
63 to oct              → 0o77
0xFF + 1               → 256
3661 to hms            → 1h 1m 1s
2.5 hr to hms          → 2h 30m 0s
100 km to mi           → ~62.14 mi
40 mi / 1 gal to km/L  → ~17.01 km/L
5 m + 300 cm to km     → 0.008 km
sin(pi / 2)            → 1
cos(0)                 → 1
sqrt(4)                → 2
log(100)               → 2
ln(e)                  → 1
abs(-5)                → 5
ceil(3.2)              → 4
floor(3.8)             → 3
round(3.5)             → 4
pow(2, 10)             → 1024
mod(10, 3)             → 1
min(3, 7)              → 3
max(3, 7)              → 7
num(5 km)              → 5
year(@2024-06-15)      → 2024
month(@2024-06-15)     → 6
day(@2024-06-15)       → 15
pi                     → 3.141592653589793
e                      → 2.718281828459045
c                      → 299792458 m/s
50%                    → 1/2
10%                    → 1/10
200 * 10%              → 20
100 C to F             → 212 F
0 C to K               → 273.15 K
32 F to C              → 0 C
1 atm to psi           → ~14.696 psi
100 W to hp            → ~0.134 hp
1 GB to MiB            → ~953.674 MiB
1 kWh to J             → 3600000 J
fv(0.05, 10, 1000)     → ~12577.89
pv(0.05, 10, 1000)     → ~7721.73
2 ** 10                → 1024
3 ** -2                → 1/9
0xFF & 0x0F            → 15
0x0F | 0xF0            → 255
0xFF ^ 0x0F            → 240
~0                     → -1
1 << 10                → 1024
1024 >> 3              → 128
5!                     → 120
10!                    → 3628800
$50 + $30              → $80.00
$100 * 1.08            → $108.00
€50                    → €50.00
50 USD                 → $50.00
50 CAD                 → 50.00 CAD
round(2.5)             → 2  (banker's rounding: half to even)
round(3.5)             → 4
```
