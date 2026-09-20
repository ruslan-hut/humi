# Hardware checklist — amazon.es

Prices and stock observed **20 Sep 2026** on amazon.es, logged in from ES.
They move; treat them as a budget, not a quote. Everything listed below was
showing as in stock with Prime or free delivery inside a week.

> **Reality check on cost.** The original plan budgeted 10–12 € per node. That is
> an AliExpress price. On amazon.es it is **~33 € of parts per node**, or
> **39–46 € all-in** once the one-time consumables are spread across 3–5 nodes.
> If the delta matters more than the delivery time, order the MCU and sensor from
> AliExpress and use Amazon only for the battery, the box and the passives.

---

## Recommended build

| # | Part | Pick | ASIN | Unit |
|---|------|------|------|------|
| 1 | MCU | **Seeed Studio XIAO ESP32C3** — 4,7★ (364) | [B0B94JZ2YF](https://www.amazon.es/dp/B0B94JZ2YF) | 14,90 € |
| 2 | Sensor | **DollaTek SHT41** Qwiic breakout — 4,4★ | [B0BKZCWYNC](https://www.amazon.es/dp/B0BKZCWYNC) | 7,99 € |
| 3 | Battery | **AFTERTECH LiPo 103450, 2000 mAh** — BMS + JST PH 2.0 — 4,4★ | [B0DND451W4](https://www.amazon.es/dp/B0DND451W4) | 7,48 € |
| 4 | Enclosure | **LeMotech ABS 80×50×26 mm**, 5-pack — 4,5★ | [B0F6YKMPB5](https://www.amazon.es/dp/B0F6YKMPB5) | 13,99 € / 5 |
| 5 | Wire | Dupont jumpers 20 cm, 120 pcs — 4,8★ | [B0FGPNMDCL](https://www.amazon.es/dp/B0FGPNMDCL) | 8,99 € / kit |

Items 4 and 5 are bought once, not per node: the box pack covers five nodes and
the jumper kit covers all of them several times over.

### Totals

Rows 4–5 of the BOM and everything under *Passives and connectors* are bought
once, not per node.

**3 nodes**

| Part | Unit | Qty | Line |
|------|------|-----|------|
| XIAO ESP32C3 | 14,90 € | 3 | 44,70 € |
| SHT41 | 7,99 € | 3 | 23,97 € |
| LiPo 2000 mAh | 7,48 € | 3 | 22,44 € |
| Boxes (5-pack) | 13,99 € | 1 | 13,99 € |
| Dupont kit | 8,99 € | 1 | 8,99 € |
| 1 MΩ ×100 | 6,50 € | 1 | 6,50 € |
| 100 nF ×200 | 6,99 € | 1 | 6,99 € |
| JST PH 2.0, 30 sets | 8,99 € | 1 | 8,99 € |
| | | | **136,57 €** |

**45,52 € per node**, with two spare boxes and consumables for a dozen more.

Add a spare MCU and sensor — recommended, you will kill one during assembly —
and it is **159,46 €**.

**5 nodes** (the box pack covers exactly five):

| Part | Unit | Qty | Line |
|------|------|-----|------|
| XIAO ESP32C3 | 14,90 € | 5 | 74,50 € |
| SHT41 | 7,99 € | 5 | 39,95 € |
| LiPo 2000 mAh | 7,48 € | 5 | 37,40 € |
| Boxes (5-pack) | 13,99 € | 1 | 13,99 € |
| Dupont kit | 8,99 € | 1 | 8,99 € |
| Passives and connectors | 22,48 € | 1 | 22,48 € |
| | | | **197,31 €** |

**39,46 € per node.** Each node past the third costs **30,37 €** marginal — box,
wire and passives are already paid for.

### Why the XIAO over the ESP32-C3 SuperMini

The SuperMini is half the price (2-pack at 14,77 €, single units from 9,85 €) but
it carries a power LED and an AMS1117 LDO that together draw more than the MCU
does asleep — boards commonly sit at 300–400 µA in deep sleep unless you lift the
LED resistor. The XIAO is specified at **43 µA deep sleep** (field reports
41–80 µA), has the Li-ion charger already on board (380 mA fast / 40 mA trickle)
and battery pads on the underside, so the TP4056 module disappears from the BOM.
At 150 µA average and a 2000 mAh cell that is roughly **8–14 months per charge** —
but see "Measure before you build" below; this is the number you verify, not the
number you trust.

Buy the SuperMini instead only if you want to spend an evening desoldering LEDs.
The XIAO ships with a u.FL whip antenna — **plug it in**, the board leans on it
for range.

---

## Alternatives, if the pick above is out of stock

**MCU**
- [Search: ESP32-C3 SuperMini](https://www.amazon.es/s?k=ESP32-C3+SuperMini) — 2-pack 14,77 €, single 9,85 €, Waveshare pre-soldered 10,90 €
- Waveshare ESP32-C6-Zero 2-pack, 17,99 € — WiFi 6 + Thread, same deep-sleep class

**Sensor** (all I²C, 3V3, same 4 wires)
- [B0DWY23XQX](https://www.amazon.es/dp/B0DWY23XQX) Adafruit SHT41 STEMMA QT — 17,93 €, 5,0★. The one to buy if you want the readings to still be right in three years.
- [B0GP1VWT53](https://www.amazon.es/dp/B0GP1VWT53) generic SHT40/41/45 module — 10,59 €
- [B0CJY2DFRP](https://www.amazon.es/dp/B0CJY2DFRP) Hailege SHT41 — 11,49 €, 3,5★
- [B0CYWMH7FC](https://www.amazon.es/dp/B0CYWMH7FC) **3× SHT31-D — 10,81 €, 4,4★**. Budget route: 3,60 €/node. ±2 % RH instead of ±1,8 %, and worse long-term drift, which for "is the cellar damp" is fine.
- [B0GSSF1G2P](https://www.amazon.es/dp/B0GSSF1G2P) 3× AHT20+BMP280 — 13,89 €. Adds barometric pressure; RH accuracy and drift are clearly worse. Only if you actually want pressure.

**Battery**
- [B09DS144QC](https://www.amazon.es/dp/B09DS144QC) EEMB 2200 mAh 773575, JST — 10,99 €
- [B0C66CNGB2](https://www.amazon.es/dp/B0C66CNGB2) LiPo 2000 mAh, JST-PH 2.0 — 12,39 €
- [B0G5PD1VRR](https://www.amazon.es/dp/B0G5PD1VRR) LiPo 2000 mAh 103450 with protection board — 15,19 €

**Enclosure** — see *Box sizing* below before you pick a variant
- [B07T6ZKJY5](https://www.amazon.es/dp/B07T6ZKJY5) Zulkit IP65 63×58×35 mm, 2-pack — gasketed, if a node goes somewhere actually wet
- [B09FXBS61N](https://www.amazon.es/dp/B09FXBS61N) 12× 60×35×25 mm — **too narrow** for a 103450 cell. USB-powered nodes only.

### Box sizing

Component footprints, in mm:

| Part | L × W × H |
|------|-----------|
| XIAO ESP32C3 | 21 × 17,5 × 3,5 |
| SHT4x breakout | ~25 × 18 × 5 |
| LiPo 103450, 2000 mAh | 50 × 34 × 10 |
| JST-PH lead, folded | ~10 of extra length |

Battery flat on the floor, boards stacked on top, slack for wire bends:

```
internal >= 62 x 38 x 16   (comfortable: 70 x 42 x 18)
```

Judge a box by its **internal** dimensions, never the external ones — these ABS
cases lose 3 mm per wall and 6 mm of height to the lid lip.

#### LeMotech 5-pack, size variants (manufacturer's own internal figures, ±2 mm)

| External | Internal | Price | Verdict |
|----------|----------|-------|---------|
| 50×28×15 | 44 × 23 × 9 | 13,54 € | No — 9 mm of height will not take a 10 mm cell, let alone the boards |
| 60×26×16 | ~54 × 21 × 10 | 12,49 € | No — 21 mm wide against a 34 mm cell |
| 72×42×23 | 66 × 37 × 17 | 12,99 € | Works, zero margin. 37 mm minus tolerance can be 35 against a 34 mm cell |
| **80×50×26** | **75 × 45 × 20** | **13,99 €** | **Buy this one** |
| 90×70×28 | ~84 × 65 × 22 | 15,49 € | Fine, just larger on the wall than it needs to be |
| 100×60×25 | 96 × 56 × 23 | 14,99 € | Fine, easiest to assemble in |

None of these are vented or gasketed — drill them (see build note 1).

If you want a genuinely small box, the battery is what you shrink, not the
enclosure: a 602535 cell (6 × 25 × 35 mm, ~500 mAh) fits the 72×42×23 variant
with room to spare, at 2–3 months of runtime per charge instead of 8–14.

---

## 18650 route (only if you already own cells and a charger)

amazon.es sells very few bare 18650 cells — most listings are packs with a
connector pre-attached, at 15–20 € each, which is worse than the LiPo above.
If you already have cells:

| Part | Pick | ASIN | Price |
|------|------|------|-------|
| Holder | TECHZOCO 18650 with leads — 4,8★ | [B0DNKTX42G](https://www.amazon.es/dp/B0DNKTX42G) | 5,75 € |
| Holder | CABLEPELADO 18650, 200 mm leads | [B07Q9W5F3L](https://www.amazon.es/dp/B07Q9W5F3L) | 5,40 € |
| Charger | 20× TP4056 USB-C with protection — 4,2★ | [B0CNGWHS5N](https://www.amazon.es/dp/B0CNGWHS5N) | 6,99 € |
| Charger | 8× TP4056 USB-C — 4,7★ | [B0H5JWWMNL](https://www.amazon.es/dp/B0H5JWWMNL) | 8,99 € |

A protected 18650 at ~3000 mAh buys ~50 % more runtime than the 2000 mAh LiPo,
at the cost of an extra module in every box.

---

## Passives and connectors

Three one-time buys that cover every node you will ever build. Needed for the
battery-sense divider and a removable cell — schematic in `firmware/README.md`.

| Part | Pick | ASIN | Price |
|------|------|------|-------|
| 1 MΩ resistor ×2/node | 100 × **1 MΩ**, 1/4 W, 1 % metal film | [B0DMV5SJWH](https://www.amazon.es/dp/B0DMV5SJWH) | 6,50 € |
| 100 nF cap ×1/node | 200 × **100 nF** (104), 50 V monolithic | [B0H9RY9RRD](https://www.amazon.es/dp/B0H9RY9RRD) | 6,99 € |
| JST PH 2.0 ×1/node | 30 sets, **male + female**, 150 mm 22 AWG red/black — 4,6★ | [B0CBWYWPB9](https://www.amazon.es/dp/B0CBWYWPB9) | 8,99 € |

**22,48 € total.** Enough for 50 nodes' worth of resistors and 30 of connectors.

Buy the JST kit with **both genders**. LiPo vendors are not consistent about
which half they crimp onto the cell, and a pigtail you solder to `B+`/`B−` is
far easier than trying to tack a through-hole header onto surface pads.

**Upgrade, if your parts drawer is empty:** swap the single-value resistor pack
for [B0CDWVLTM1](https://www.amazon.es/dp/B0CDWVLTM1) — 50 values, 0 Ω to 10 MΩ,
1 % metal film, 1 MΩ confirmed in the value list, 10,99 €, 4,6★. Costs 4,49 €
more and you stop ordering resistors one value at a time.

**Heat-shrink: skip it.** Every connection in this node is either
vendor-crimped and housed (the cell's JST plug, the Dupont jumpers) or a wire
soldered flat to an SMD pad, which shrink cannot cover anyway. The only bare
metal is the divider's axial leads — sleeve those with a few cm from the drawer,
or build the divider on a 10 × 10 mm scrap of perfboard and the problem is gone.
A 560-piece assortment is not worth ordering for three nodes.

What does earn its place, if you have neither:

| Part | Why | ASIN | Price |
|------|-----|------|-------|
| Kapton tape 10 mm | Insulates the cell face and gives the battery leads strain relief | [B0G263T6GR](https://www.amazon.es/dp/B0G263T6GR) 4,8★ | 6,19 € |
| Double-sided foam tape | Mounts the boards and keeps component leads off the pouch | [B0C7CPRQYH](https://www.amazon.es/dp/B0C7CPRQYH) 4,2★ | 8,80 € |

Cheap electrical tape and a blob of hot glue do the same job.

---

## Tools you need at least once

- Soldering iron, thin solder, flux — two joints per node (battery pads), four
  more if you solder the sensor instead of using Dupont headers
- Tape — Kapton, electrical or foam. Not heat-shrink; see *Passives and
  connectors*.
- **A multimeter that reads µA.** A USB power meter cannot see 100 µA and will
  tell you nothing. Without a µA range you are guessing at battery life; a
  Nordic PPK2 is the nice version if you plan to keep doing this.
- USB-C cable for flashing

---

## Build notes that cost a battery if ignored

1. **Vent the box.** A sealed enclosure measures its own interior, not the room:
   the reading lags hours behind and reads high. Drill 4–6 holes of 3 mm behind
   the sensor, or mount the sensor head outside the box entirely.

2. **Power-gate the sensor module, not the sensor.** The SHT4x itself idles at
   0,08 µA — gating it saves nothing. What you are switching off is the
   breakout's **power LED** (1–3 mA, i.e. 20× your entire sleep budget) and the
   I²C pull-ups. Cheap modules without an LED can stay on 3V3.

3. **Cache the BSSID and channel in RTC memory.** A cold WiFi scan costs 3–6 s at
   ~100 mA; a targeted reconnect is well under a second. This single change is
   worth more than every other optimisation combined.

4. **Two upload attempts, then sleep.** An unreachable server plus an infinite
   retry loop flattens a cell overnight. Buffer the reading and send it on the
   next wake with the `age_s` field — the API takes a batch.

5. **Measure before you build three.** Assemble one node, put the meter in
   series, read the deep-sleep current, and only then decide the interval.
   Expected: 40–80 µA asleep, ~80 mA for the 1–3 s awake window.
   `average = sleep + awake_mA × awake_s / interval_s`.
