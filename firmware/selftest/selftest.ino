// Bench self-test for a XIAO ESP32C3: chip info, WiFi scan, I2C scan and a
// live SHT4x reading. Flash it onto every board and sensor before assembly;
// nothing here is kept in the node firmware.
#include <WiFi.h>
#include <Wire.h>

const uint8_t SHT4X_ADDR = 0x44;
const uint8_t SHT4X_MEASURE_HIGH = 0xFD;  // high precision, ~8,3 ms

const unsigned long SCAN_EVERY_MS = 30000;
const unsigned long READ_EVERY_MS = 2000;

unsigned long lastScan = 0;

// CRC-8, polynomial 0x31, init 0xFF, as specified in the SHT4x datasheet.
uint8_t crc8(const uint8_t* data) {
  uint8_t crc = 0xFF;
  for (int i = 0; i < 2; i++) {
    crc ^= data[i];
    for (int bit = 0; bit < 8; bit++) {
      crc = (crc & 0x80) ? (crc << 1) ^ 0x31 : crc << 1;
    }
  }
  return crc;
}

bool readSht4x(float& temp, float& rh) {
  Wire.beginTransmission(SHT4X_ADDR);
  Wire.write(SHT4X_MEASURE_HIGH);
  if (Wire.endTransmission() != 0) return false;
  delay(10);

  uint8_t b[6];
  if (Wire.requestFrom(SHT4X_ADDR, (uint8_t)6) != 6) return false;
  for (int i = 0; i < 6; i++) b[i] = Wire.read();
  if (crc8(b) != b[2] || crc8(b + 3) != b[5]) return false;

  uint16_t rawT = (b[0] << 8) | b[1];
  uint16_t rawRH = (b[3] << 8) | b[4];
  temp = -45.0f + 175.0f * rawT / 65535.0f;
  rh = constrain(-6.0f + 125.0f * rawRH / 65535.0f, 0.0f, 100.0f);
  return true;
}

void scan() {
  Serial.println();
  Serial.printf("== chip %s rev %d, MAC %s, uptime %lus\n",
                ESP.getChipModel(), ESP.getChipRevision(),
                WiFi.macAddress().c_str(), millis() / 1000);

  int n = WiFi.scanNetworks();
  Serial.printf("wifi: %d networks\n", n);
  for (int i = 0; i < n && i < 8; i++) {
    Serial.printf("  %4d dBm  ch%-2d  %s\n", WiFi.RSSI(i), WiFi.channel(i), WiFi.SSID(i).c_str());
  }
  WiFi.scanDelete();

  int found = 0;
  for (uint8_t addr = 1; addr < 127; addr++) {
    Wire.beginTransmission(addr);
    if (Wire.endTransmission() == 0) {
      Serial.printf("i2c: device at 0x%02X%s\n", addr, addr == SHT4X_ADDR ? "  <- SHT4x" : "");
      found++;
    }
  }
  if (found == 0) Serial.println("i2c: no devices");
  Serial.println();
}

void setup() {
  Serial.begin(115200);
  delay(2000);  // give the Mac time to reopen the USB serial port

  Wire.begin(SDA, SCL);  // D4 = GPIO6, D5 = GPIO7
  WiFi.mode(WIFI_STA);
  scan();
  lastScan = millis();
}

void loop() {
  if (millis() - lastScan >= SCAN_EVERY_MS) {
    scan();
    lastScan = millis();
  }

  float temp, rh;
  if (readSht4x(temp, rh)) {
    Serial.printf("sht4x: %5.1f %%RH  %5.2f C\n", rh, temp);
  } else {
    Serial.println("sht4x: read failed");
  }
  delay(READ_EVERY_MS);
}
