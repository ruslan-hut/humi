// Minimal SHT4x driver: one high-precision measurement, CRC-checked.
#pragma once
#include <Wire.h>

namespace sht4x {

const uint8_t ADDR = 0x44;
const uint8_t MEASURE_HIGH = 0xFD;  // ~8,3 ms

// CRC-8, polynomial 0x31, init 0xFF, as specified in the SHT4x datasheet.
inline uint8_t crc8(const uint8_t* data) {
  uint8_t crc = 0xFF;
  for (int i = 0; i < 2; i++) {
    crc ^= data[i];
    for (int bit = 0; bit < 8; bit++) {
      crc = (crc & 0x80) ? (crc << 1) ^ 0x31 : crc << 1;
    }
  }
  return crc;
}

inline bool read(float& temp, float& rh) {
  Wire.beginTransmission(ADDR);
  Wire.write(MEASURE_HIGH);
  if (Wire.endTransmission() != 0) return false;
  delay(10);

  uint8_t b[6];
  if (Wire.requestFrom(ADDR, (uint8_t)6) != 6) return false;
  for (int i = 0; i < 6; i++) b[i] = Wire.read();
  if (crc8(b) != b[2] || crc8(b + 3) != b[5]) return false;

  uint16_t rawT = (b[0] << 8) | b[1];
  uint16_t rawRH = (b[3] << 8) | b[4];
  temp = -45.0f + 175.0f * rawT / 65535.0f;
  rh = constrain(-6.0f + 125.0f * rawRH / 65535.0f, 0.0f, 100.0f);
  return true;
}

}  // namespace sht4x
