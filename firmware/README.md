# Firmware

ESP32-C3, Arduino framework. Not written yet — see section 6 of
`../IMPLEMENTATION.md` for the wake cycle and the RTC-memory buffer.

---

## Schematic

Three blocks. The XIAO is the only active part; the sensor hangs off I²C and the
cell hangs off the charger that is already on the board.

```
        ┌──────────────┐
        │  LiPo 3,7 V  │
        │  2000 mAh    │  JST PH 2.0, BMS on the cell
        │  103450      │
        └──┬────────┬──┘
           │ +      │ −
           │        │
        ┌──▼────────▼───────────────────────────┐
        │  B+      B−     (pads, underside)     │
        │                                       │
        │        XIAO ESP32C3                   │──● u.FL ── whip antenna
        │                                       │
        │  3V3   GND   D4      D5       D2      │
        └───┬─────┬─────┬───────┬────────┬──────┘
            │     │     │       │        │
            │     │     │ SDA   │ SCL    │ A2 / ADC1_CH4
            │     │     │       │        │
        ┌───▼─────▼─────▼───────▼──┐     │
        │ VIN   GND   SDA    SCL   │     │
        │      SHT41 breakout      │     │
        └──────────────────────────┘     │
                                         │
              B+ ──[ R1 1M ]──┬──────────┘
                              │
                            [ R2 1M ]   [ C1 100n ]
                              │             │
              GND ────────────┴─────────────┘
```

## Net list

| Net | XIAO pin | GPIO | Goes to |
|-----|----------|------|---------|
| VDD | `3V3` | — | SHT41 `VIN` |
| GND | `GND` | — | SHT41 `GND`, divider bottom |
| I²C data | `D4` | GPIO6 | SHT41 `SDA` |
| I²C clock | `D5` | GPIO7 | SHT41 `SCL` |
| Battery sense | `D2` / `A2` | GPIO4 | divider mid-point |
| Battery | `B+` / `B−` pads | — | cell, via a JST PH 2.0 pigtail |

Four wires to the sensor, two to the battery, two resistors and a capacitor.
That is the whole node.

## Battery

The XIAO charges the cell itself over USB-C — 380 mA fast, 40 mA trickle. No
TP4056.

**Solder a JST PH 2.0 *socket* to the `B+`/`B−` pads, not the cell directly.**
A removable cell means you can flash and debug on USB with the battery out, and
you are not desoldering a charged LiPo when one dies.

**Check the polarity with a meter before the first plug-in.** JST PH pigtails
are not wired consistently between vendors — the AFTERTECH listing says so
itself. Red to `B+`, black to `B−`; reversed kills the board.

The cell has its own BMS, which is what stops the ESP32 from dragging it below
2,5 V and wrecking it. Do not substitute a bare cell without one.

## Battery sense divider

The `B+` pad is not connected to any GPIO on the XIAO ESP32C3 — if you want
`vbat` in the payload you add the divider yourself.

- **R1 = R2 = 1 MΩ**, mid-point to `D2`. Reads VBAT/2: 2,10 V full, 1,50 V empty,
  comfortably inside the ADC's ~2500 mV full scale at 12 dB attenuation.
- **C1 = 100 nF** across R2. The ESP32 ADC wants a low-impedance source; without
  the cap a 500 kΩ source gives you noise. Throw away the first sample after
  wake anyway.
- Use `analogReadMilliVolts(A2)` and multiply by 2 — the C3 carries ADC
  calibration in efuse, so no hand-calibration is needed.

Seeed's own examples use 200 kΩ or even 10 kΩ. Don't. A 200 k/200 k divider
draws 10,5 µA continuously, a quarter of the node's entire sleep budget; 10 k/10 k
draws 210 µA and flattens the cell in a fortnight. 1 MΩ costs 2,1 µA, about 5 %.

**Use `D2`/`A2`, not `D0`/`A0`.** GPIO2 is a strapping pin on the ESP32-C3 — a
divider holding it mid-rail at boot can stop the board coming up. GPIO4 has no
such role. Avoid `D8` and `D9` for the same reason.

## Sensor power gating — probably not needed

The SHT41 itself idles at **80 nA** and averages 0,4 µA at 1 Hz. Gating that
saves nothing.

Gate it only if **your breakout has a power LED**, which costs 1–3 mA — twenty
times the node's entire sleep budget. Check the board under a light with USB
attached. If there is one:

- cut its jumper or lift the resistor, which is the better fix, or
- move SHT41 `VIN` from `3V3` to **`D1` / GPIO3**, drive it HIGH on wake, LOW
  before sleep. GPIO3 is RTC-capable so `gpio_hold_en()` keeps it low through
  deep sleep, and it is not a strapping pin.

If you gate it, also `Wire.end()` and set `D4`/`D5` to `INPUT` before sleeping,
or the ESP32 back-feeds the unpowered module through its ESD diodes.

The module's I²C pull-ups leak nothing while the bus idles high, so an
LED-less board can stay on `3V3` permanently.

## Pin states before `esp_deep_sleep_start()`

| Pin | State |
|-----|-------|
| `D4`, `D5` | `INPUT` after `Wire.end()` |
| `D2` | `INPUT` — the divider is passive, nothing to switch |
| `D1` | `LOW` + `gpio_hold_en()`, only if the gate is fitted |

## Node credentials

`nodectl` prints a token once at registration. Store it in the sketch next to the
WiFi credentials; it is the only thing authenticating the node.

```cpp
const char* HUMI_URL   = "https://humi.example.com/api/v1/readings";
const char* HUMI_TOKEN = "…";   // from: nodectl -slug bedroom -name Bedroom
```

## Extra parts the BOM does not cover

Two 1 MΩ resistors, one 100 nF capacitor and a JST PH 2.0 socket per node — a
few cents if you have a parts drawer, otherwise add an assortment kit to the
order.
