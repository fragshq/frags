# dates

Date manipulation utilities for frags' evaluation and scripting engines:

* **Go templates** (`evaluators.EvaluateTemplate`), e.g. `{{ dateFormat (now) "DateOnly" }}`
* **expr expressions** (`evaluators.EvaluateExpression` and friends, `$(...)` references), e.g. `dateFormat(now(), "DateOnly")`
* **JavaScript scripts** (`scriptengines.JavascriptScriptingEngine`), e.g. `dateFormat(now(), "DateOnly")`

The same three functions are available in all engines, with **the same names, argument order and optional
arguments**.

| Function     | Signature                       | Returns                |
|--------------|---------------------------------|------------------------|
| `now`        | `now(tz?)`                      | the current time       |
| `dateFormat` | `dateFormat(t, layout, tz?)`    | a formatted string     |
| `dateAdd`    | `dateAdd(t, duration)`          | the shifted time       |

`?` marks an optional argument.

---

## `now(tz?)`

Returns the current time.

* `tz` (optional): an IANA timezone name (`"Europe/Rome"`, `"America/New_York"`), `"UTC"` or `"Local"`
  (the timezone of the machine running frags). **Defaults to UTC.**

```
{{ now }}                      -> 2026-11-09T10:15:00Z
{{ now "Europe/Rome" }}        -> 2026-11-09T11:15:00+01:00
```
```
now()
now("Europe/Rome")
```

## `dateFormat(t, layout, tz?)`

Formats a time.

* `t`: a time value (see [Accepted time values](#accepted-time-values)).
* `layout`: a [Go layout](https://pkg.go.dev/time#pkg-constants), a preset name, or a special layout
  (see [Layouts](#layouts)).
* `tz` (optional): converts the time to this timezone before formatting. **If omitted, the time keeps its own
  timezone** (so `dateFormat(now("Europe/Rome"), "DateTime")` prints Rome time).

```
{{ dateFormat .params.ts "DateOnly" }}                    -> 2026-03-28
{{ dateFormat .params.ts "02/01/2006 15:04" }}            -> 28/03/2026 10:30
{{ dateFormat .params.ts "DateTime" "Europe/Rome" }}      -> 2026-03-28 11:30:00
{{ dateFormat .params.ts "unix" }}                        -> 1774693800
```
```
dateFormat(params.ts, "DateOnly")
dateFormat(params.ts, "DateTime", "Europe/Rome")
```

## `dateAdd(t, duration)`

Adds a duration to a time. Prefix the duration with `-` to subtract.

* `t`: a time value (see [Accepted time values](#accepted-time-values)).
* `duration`: a duration string (see [Durations](#durations)).

The result keeps the timezone of the input.

```
{{ dateAdd .params.ts "1d" }}                             -> 2026-03-29T10:30:00Z
{{ dateFormat (dateAdd (now) "-7d") "DateOnly" }}         -> one week ago
```
```
dateAdd(params.ts, "1d")
dateFormat(dateAdd(now(), "-7d"), "DateOnly")
now() | dateAdd("-1w") | dateFormat("DateOnly")
```

---

## Accepted time values

Wherever a function takes a time `t`, it accepts:

| Input                               | Example                                              |
|-------------------------------------|------------------------------------------------------|
| a time returned by `now` / `dateAdd`| `dateAdd(now(), "1h")`                               |
| an RFC3339 string                   | `"2026-03-28T10:30:00Z"`, `"2026-03-28T11:30:00+01:00"` |
| a date-time string (UTC)            | `"2026-03-28 10:30:00"`                              |
| a date string (midnight UTC)        | `"2026-03-28"`                                       |
| a number: Unix timestamp in seconds | `1774693800`, `1774693800.5`                         |

This means values coming from params, context or LLM output can be passed directly, without parsing them first.
Any other value is an error.

## Layouts

`layout` can be:

1. **A Go layout**, built from Go's reference time `Mon Jan 2 15:04:05 MST 2006`, e.g. `"2006-01-02"`,
   `"02/01/2006 15:04"`, `"Monday, January 2"`. See [the Go docs](https://pkg.go.dev/time#Layout).
2. **A preset name**:

   | Preset        | Output example                        |
   |---------------|---------------------------------------|
   | `DateOnly`    | `2026-03-28`                          |
   | `DateTime`    | `2026-03-28 10:30:00`                 |
   | `TimeOnly`    | `10:30:00`                            |
   | `RFC3339`     | `2026-03-28T10:30:00Z`                |
   | `RFC3339Nano` | `2026-03-28T10:30:00.123456789Z`      |
   | `RFC1123`     | `Sat, 28 Mar 2026 10:30:00 UTC`       |
   | `RFC1123Z`    | `Sat, 28 Mar 2026 10:30:00 +0000`     |
   | `RFC822`      | `28 Mar 26 10:30 UTC`                 |
   | `RFC822Z`     | `28 Mar 26 10:30 +0000`               |
   | `RFC850`      | `Saturday, 28-Mar-26 10:30:00 UTC`    |
   | `ANSIC`       | `Sat Mar 28 10:30:00 2026`            |
   | `UnixDate`    | `Sat Mar 28 10:30:00 UTC 2026`        |
   | `RubyDate`    | `Sat Mar 28 10:30:00 +0000 2026`      |
   | `Kitchen`     | `10:30AM`                             |
   | `Stamp`       | `Mar 28 10:30:00`                     |
   | `StampMilli`  | `Mar 28 10:30:00.000`                 |
   | `StampMicro`  | `Mar 28 10:30:00.000000`              |
   | `StampNano`   | `Mar 28 10:30:00.000000000`           |

3. **A special layout**:

   | Layout   | Output                                       |
   |----------|----------------------------------------------|
   | `unix`   | Unix timestamp in seconds, e.g. `1774693800` |
   | `unixms` | Unix timestamp in milliseconds               |

Preset names are case-sensitive.

## Durations

Durations use the [Go duration syntax](https://pkg.go.dev/time#ParseDuration), with two extra units:

| Unit       | Meaning                      |
|------------|------------------------------|
| `ns`       | nanoseconds                  |
| `us` / `µs`| microseconds                 |
| `ms`       | milliseconds                 |
| `s`        | seconds                      |
| `m`        | minutes                      |
| `h`        | hours                        |
| `d`        | days (integer only)          |
| `w`        | weeks (integer only)         |

* Units can be combined: `"1w2d3h30m"`.
* A leading `-` subtracts: `"-1d12h"` is minus 36 hours. A leading `+` is allowed.
* `"0"` is a valid zero duration. Any other number needs a unit.
* Fractions are allowed for units other than days and weeks: `"1.5h"` works, `"1.5d"` doesn't (use `"1d12h"`).

**Days and weeks follow the calendar.** `dateAdd(t, "1d")` returns the same wall-clock time on the next day,
even across a daylight saving change, when that day is 23 or 25 hours long. `"24h"` always adds exactly 24
hours. This only matters for times in a timezone with DST; in UTC the two are the same.

## Timezones

* Timezone names are [IANA names](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones), plus `"UTC"`
  and `"Local"`.
* The timezone database is embedded in the binary (`time/tzdata`), so names resolve on every platform,
  including Windows, without depending on the system's zone files.
* An unknown name is an error.

## Differences between the two engines

The functions have the same names and arguments in both engines. There are three differences, each due to how
the engine works:

### 1. Return type of `now` and `dateAdd`

* **Templates** get a `dates.Time`, which wraps `time.Time` and **prints as RFC3339** when rendered without
  formatting: `{{ now }}` renders `2026-11-09T10:15:00Z`, not Go's default
  `2026-11-09 10:15:00.123 +0000 UTC m=+0.0004`. All `time.Time` methods remain available, e.g.
  `{{ (now).Unix }}` or `{{ (now).Weekday }}`. `{{ json (now) }}` produces an RFC3339 string that keeps
  fractional seconds (`"2026-11-09T10:15:00.123456Z"`), as with any Go `time.Time`.
* **expr** gets a plain `time.Time`, so that it works with expr's own time support: comparisons
  (`now() > date("2026-01-01")`), subtraction (`t1 - t2` returns a duration) and methods
  (`now().Year()`).

### 2. Calling without arguments

In templates, `now` can be used bare or in parentheses: `{{ now }}`, `{{ dateAdd now "1d" }}`,
`{{ dateAdd (now "UTC") "1d" }}`. In expr it always needs parentheses: `now()`.

### 3. Pipes

The two engines' pipes pass the value in opposite positions:

* **expr** passes the piped value as the **first** argument, so pipes work naturally:
  `now() | dateAdd("1d") | dateFormat("DateOnly")`
* **Go templates** pass the piped value as the **last** argument, so `{{ now | dateFormat "DateOnly" }}`
  does **not** work. Nest with parentheses instead: `{{ dateFormat (dateAdd now "1d") "DateOnly" }}`

## JavaScript

In JavaScript scripts the functions are plain globals, called as in expr:

```js
const lastWeek = dateAdd(now(), "-1w");
const label = dateFormat(lastWeek, "DateOnly", "Europe/Rome");
```

JS-specific behaviour:

* **`now` and `dateAdd` return native JS `Date` objects**, so all the usual `Date` methods and operators work:
  `now().getTime()`, `dateAdd(t, "1h").toISOString()`, `dateAdd(t, "1h") - new Date(t)` (milliseconds),
  `a < b`.
* **`Date` objects are accepted as input**, in addition to the [accepted time values](#accepted-time-values).
* **JS `Date`s have no timezone.** So `now("Europe/Rome")` returns the same instant as `now()`, and when a `Date`
  is passed to `dateFormat` without `tz` it is formatted in **UTC** (never in the host's local zone). To show a
  local time, pass the timezone to `dateFormat`: `dateFormat(now(), "DateTime", "Europe/Rome")`.
  For the same reason, `dateAdd` on a `Date` does its day arithmetic in UTC.
* **Numbers are Unix timestamps in seconds**, as in the other engines, **not milliseconds** as is usual in JS.
  Pass a `Date` rather than a raw `Date.now()` / `getTime()` value: `dateFormat(new Date(ms), ...)`.
* **Errors are thrown as `TypeError`s**, so they can be caught with `try/catch`. An uncaught error fails the
  script.

## Interaction with expr built-ins

expr ships built-in time functions. `now()` is overridden by the version described here: it returns the same
kind of value but adds the optional timezone and returns UTC by default. The other built-ins, such as `date()` and
`duration()`, are untouched and can be mixed freely with these functions:

```
dateAdd(params.ts, "1d") > date("2026-03-28T12:00:00Z")
dateAdd(params.ts, "1h") - date(params.ts)        // 1h0m0s
```

## Go API

The package can also be used directly from Go:

```go
dates.Now(tz string) (time.Time, error)                      // "" = UTC
dates.Format(t any, layout string, tz string) (string, error) // "" = keep t's timezone
dates.Add(t any, duration string) (time.Time, error)
dates.ParseDuration(s string) (time.Duration, error)          // days are flat 24h here
dates.ToTime(v any) (time.Time, error)                        // conversion rules above
dates.LoadLocation(tz string) (*time.Location, error)         // "" = UTC
dates.Time                                                    // time.Time that prints as RFC3339
```
