# Firmware

ESP32-C3, Arduino framework, built and flashed with `arduino-cli`.

- `selftest/` — chip info, WiFi scan, I²C scan and a live SHT4x reading.
  Flash it onto every board and sensor before assembly.
- `node/` — the node. Currently the **bench build** (phase 2): USB power, no
  deep sleep, POST every `interval_s`. The wake cycle and RTC-memory buffer of
  section 6 of `../IMPLEMENTATION.md` come next.

## Build and flash

```sh
brew install arduino-cli
arduino-cli config add board_manager.additional_urls \
  https://raw.githubusercontent.com/espressif/arduino-esp32/gh-pages/package_esp32_index.json
arduino-cli core update-index && arduino-cli core install esp32:esp32

cp node/secrets.example.h node/secrets.h     # WiFi, server URL, node token
arduino-cli compile -b esp32:esp32:XIAO_ESP32C3 node
arduino-cli upload  -b esp32:esp32:XIAO_ESP32C3 -p /dev/cu.usbmodem1101 node
```

Serial output is 115200 baud on the same USB port. The node collects its log
and prints it just before sleeping, and only waits for a host when USB is
plugged in, so the wake log survives the port re-enumerating.

**Flashing.** From 0.3.0 the node does not deep-sleep while a USB *host* is
attached (it waits awake between readings instead), so plugging it into the
Mac and waiting for the next wake is enough. A USB charger has no host and
does not change anything. For older firmware, or a crashed one: the node is
awake for well under a second, too short for `upload` to catch. Hold **B**, tap **R**, release **B**: the ROM bootloader
enumerates and waits. After flashing, the RTS reset leaves it in the
bootloader; tap **R** again, or reset it from the Mac:

```sh
esptool --chip esp32c3 -p /dev/cu.usbmodem1101 --before no-reset --after watchdog-reset chip-id
```

For the bench, point `HUMI_URL` at the machine running the server and bind the
server to its LAN address in a gitignored `config.local.yml`.

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

## Headers

The XIAO ships with loose header strips. **Do not solder `5V`, `D0` or `D1`** —
they sit against the USB-C connector, and heat there cracked the connector's
own joints on the first board: USB went intermittent, then dead. The node uses
none of them. Solder `GND` last, one quick touch from the board edge.

Solder the far pins too (`D6`, `D7`–`D10`) even though they are unused: they
hold the strips when Dupont leads are pulled off.

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
- Use `analogReadMilliVolts(A2)` and multiply by 2. The C3 carries ADC
  calibration in efuse, but resistor tolerance still shows: the bench node read
  2,1 % high. Measure the cell with a meter **on battery, not while charging**,
  and put `meter / reported` into `VBAT_SCALE` in that node's `secrets.h`.
- **Do not check the divider with the meter's voltage range.** A meter with
  ~1 MΩ input impedance (the ANENG 681 in auto mode) sits in parallel with R2
  and reads 1,25 V where the ADC sees 2,0 V. Compare `vbat` against the cell
  voltage instead.

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
- move SHT41 `VIN` from `3V3` to **`D3` / GPIO5**, drive it HIGH on wake, LOW
  before sleep. GPIO5 is RTC-capable so `gpio_hold_en()` keeps it low through
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
| `D3` | `LOW` + `gpio_hold_en()`, only if the gate is fitted |

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
