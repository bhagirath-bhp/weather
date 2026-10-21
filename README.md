# wind

Live 10 m wind over a draggable globe, from NOAA GDAS 0.25° analyses.
**Demo:** https://weather-ruddy-three.vercel.app/

A small Go service on Vercel fetches the latest GDAS cycle from NOMADS,
pulls out the 10 m U/V components, and ships the full 1440×721 grid to a
plain-Canvas frontend that advects ~27k particles through it.

## How it works

```
browser ──GET /weather/current──▶ Vercel rewrite ──▶ api/weather.go
                                                       │
                                     walk back ≤8 GDAS cycles (00/06/12/18z)
                                                       │
                                     NOMADS filter_gdas_0p25.pl
                                     (server-side subset: UGRD+VGRD, 10 m, f000)
                                                       │
                                     nilsmagnus/grib → Section 7 data
                                                       │
                                     int16 ×100, row deltas, gzip ──▶ ~2.3 MB
                                                       │
browser: Int16Array → Float32Array ◀───────────────────┘
         bilinear sample, particle advection, orthographic projection
```

**Data.** GDAS is the analysis (f000) that initialises GFS; it lags real
time by 3–6 h, so `/weather/current` walks back up to eight 6-hourly cycles
until one downloads and parses. `/weather/YYYYMMDD` serves the 00z cycle
of a given day. GRIB2 parsing is delegated to
[nilsmagnus/grib](https://github.com/nilsmagnus/grib); this repo does not
implement a GRIB2 decoder.

**Why a binary payload.** Vercel serverless responses are capped at
4.5 MB. The full 0.25° grid is 1,038,240 points × 2 components; as float64
JSON that is ~25 MB, which is why the first version decimated to a 1° grid
(`?stride=4`). Packing each component as int16 at 0.01 m/s resolution gives
4.15 MB raw, under the limit before compression. Gzip alone only gets that
to ~3.0 MB because 0.01 m/s values are noisy in the low bits; delta-coding
each row first (value minus its west neighbour, in wrapping int16
arithmetic) makes the deltas small and gzip takes it to ~2.3 MB. Measured
on the 2026-04-20 00z cycle; decode is bit-exact. Wire format
(little-endian):

```
"WND2"   4 bytes   magic + version
hlen     uint32    JSON header length (space-padded to even)
header   hlen      {"refTime","nx","ny","lo1","la1","dx","dy"}
u        nx*ny     int16 row deltas, row-major, north→south, west→east
v        nx*ny     int16 row deltas
```

`?format=json` returns the old float64 JSON; `?stride=N` (1–16) decimates
either format.

**Frontend.** No framework, no d3. Orthographic projection and its inverse
are written out by hand so drag-to-rotate and wheel-zoom stay cheap.
Particles are sampled bilinearly with longitude wrap, advected with a
cos(lat) correction, and drawn as one `Path2D` per speed bucket so a frame
costs seven `stroke()` calls rather than 27k. Trails fade with a
`destination-in` fill. Borders come from `world-atlas` via `topojson-client`.

## Run locally

```sh
MET_DIR=./_ignore_/metfiles go run .      # http://localhost:8080
```

`MET_DIR` is where downloaded GRIB files are cached. On Vercel it is `/tmp`,
which is per-instance and ephemeral; the HTTP `Cache-Control` header (10 min
for `current`, 1 h for dated cycles) does the real caching at the edge.

## Known limits

- Analysis only; no forecast hours yet (GFS `f003…f384` is the obvious next step).
- 10 m level only.
- NOMADS rate-limits aggressively; a cold cache on a fresh instance can take several seconds.
