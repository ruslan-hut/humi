// humi node, bench build (phase 2): USB power, no deep sleep.
// Reads the SHT4x, POSTs to /api/v1/readings, waits interval_s from the reply.
// Deep sleep, the RTC buffer and the battery divider come after this path is
// proven end to end — see IMPLEMENTATION.md section 6.
#include <WiFi.h>
#include <HTTPClient.h>
#include <Wire.h>

#include "secrets.h"
#include "sht4x.h"

const char* FW_VERSION = "0.1.0-bench";

const unsigned long WIFI_TIMEOUT_MS = 15000;
const int DEFAULT_INTERVAL_S = 30;

int intervalS = DEFAULT_INTERVAL_S;

bool connectWifi() {
  if (WiFi.status() == WL_CONNECTED) return true;

  Serial.printf("wifi: connecting to %s\n", WIFI_SSID);
  WiFi.begin(WIFI_SSID, WIFI_PASS);
  unsigned long start = millis();
  while (WiFi.status() != WL_CONNECTED) {
    if (millis() - start > WIFI_TIMEOUT_MS) {
      Serial.printf("wifi: timeout, status %d\n", WiFi.status());
      WiFi.disconnect();
      return false;
    }
    delay(100);
  }
  Serial.printf("wifi: connected in %lu ms, ip %s, rssi %d dBm\n",
                millis() - start, WiFi.localIP().toString().c_str(), WiFi.RSSI());
  return true;
}

// Returns the HTTP status, or a negative HTTPClient error.
int post(float rh, float temp) {
  char body[128];
  snprintf(body, sizeof(body),
           "{\"rh\":%.2f,\"temp\":%.2f,\"rssi\":%d,\"fw\":\"%s\"}",
           rh, temp, WiFi.RSSI(), FW_VERSION);

  HTTPClient http;
  http.setTimeout(5000);
  http.begin(HUMI_URL);
  http.addHeader("Content-Type", "application/json");
  http.addHeader("Authorization", "Bearer " HUMI_TOKEN);

  int code = http.POST(body);
  String reply = code > 0 ? http.getString() : http.errorToString(code);
  http.end();

  Serial.printf("post: %s -> %d %s\n", body, code, reply.c_str());

  // The server owns the cadence; pick up interval_s without a JSON library.
  int at = reply.indexOf("\"interval_s\":");
  if (code == 200 && at >= 0) {
    int s = reply.substring(at + 13).toInt();
    if (s >= 10 && s <= 86400) intervalS = s;
  }
  return code;
}

void setup() {
  Serial.begin(115200);
  delay(2000);  // give the Mac time to reopen the USB serial port
  Wire.begin(SDA, SCL);  // D4 = GPIO6, D5 = GPIO7
  WiFi.mode(WIFI_STA);   // before macAddress(), which reads 0s until the driver is up

  Serial.printf("\nhumi node %s, MAC %s\n", FW_VERSION, WiFi.macAddress().c_str());
}

void loop() {
  float temp, rh;
  if (!sht4x::read(temp, rh)) {
    Serial.println("sht4x: read failed");
  } else if (connectWifi()) {
    // Two attempts, then give up until the next interval (build note 4).
    if (post(rh, temp) <= 0) post(rh, temp);
  }

  Serial.printf("sleep: %d s\n\n", intervalS);
  delay(intervalS * 1000UL);
}
