// humi node: wake, measure, POST, deep sleep. See IMPLEMENTATION.md section 6.
//
// Everything that must survive deep sleep lives in RTC memory: the cadence the
// server asked for, the access point to reconnect to without a scan, and the
// readings that could not be sent yet.
#include <WiFi.h>
#include <HTTPClient.h>
#include <WiFiClientSecure.h>
#include <Wire.h>
#include <sys/time.h>
#include "driver/gpio.h"
#include "esp_sleep.h"

#include "ca.h"
#include "secrets.h"
#include "sht4x.h"

const char* FW_VERSION = "0.4.0";

// The sensor module is powered from a GPIO, so its LED, LDO and pull-ups are
// off while the node sleeps. -1 when its VIN is wired to 3V3 instead.
const int SENSOR_POWER_PIN = D3;  // GPIO5: RTC-capable, not a strapping pin

// 1M/1M divider from BAT+ to D2. Leave false until it is fitted: a floating
// ADC pin reads noise, and the server would store it as battery voltage.
const bool HAS_VBAT_DIVIDER = true;
const int VBAT_PIN = A2;

// Per-node correction for resistor tolerance and ADC gain: meter reading
// divided by what the node reports, taken on battery, not while charging.
// Set it in secrets.h, which is per node anyway.
#ifndef VBAT_SCALE
#define VBAT_SCALE 1.0f
#endif

const unsigned long WIFI_TIMEOUT_MS = 10000;
const int DEFAULT_INTERVAL_S = 900;
const int FULL_SCAN_AFTER_FAILURES = 5;
const int BUFFER_SIZE = 8;

struct Sample {
  float rh;
  float temp;
  float vbat;
  time_t takenAt;  // RTC seconds since power-on; the clock keeps running in deep sleep
};

// DHCP costs ~3 s of radio time per wake, more than everything else together.
// Reuse the last lease for a while; a fresh DHCP round renews it with the router.
const time_t LEASE_REUSE_S = 6 * 3600;

RTC_DATA_ATTR int intervalS = DEFAULT_INTERVAL_S;
RTC_DATA_ATTR uint8_t apBssid[6];
RTC_DATA_ATTR int apChannel = 0;  // 0 = unknown, do a full scan
RTC_DATA_ATTR uint32_t leaseIp = 0, leaseGw = 0, leaseMask = 0, leaseDns = 0;  // 0 = none, use DHCP
RTC_DATA_ATTR time_t leaseAt = 0;
RTC_DATA_ATTR int failures = 0;
RTC_DATA_ATTR Sample buffer[BUFFER_SIZE];
RTC_DATA_ATTR int buffered = 0;

// The USB port only reappears on the Mac a second or so after each wake, so
// anything printed straight away is lost. Collect the log and print it just
// before sleeping, waiting briefly for the host only when USB is plugged in.
String logBuf;

void logf(const char* fmt, ...) {
  char line[512];
  va_list args;
  va_start(args, fmt);
  vsnprintf(line, sizeof(line), fmt, args);
  va_end(args);
  logBuf += line;
}

void flushLog() {
  if (!HWCDC::isPlugged()) return;  // on battery: no host, no wait
  unsigned long start = millis();
  while (!HWCDC::isConnected() && millis() - start < 2000) delay(10);
  Serial.setTxTimeoutMs(200);  // the host is there now; let the log drain
  Serial.print(logBuf);
  Serial.flush();
}

time_t nowS() {
  struct timeval tv;
  gettimeofday(&tv, nullptr);
  return tv.tv_sec;
}

void sensorPower(bool on) {
  if (SENSOR_POWER_PIN < 0) return;
  gpio_hold_dis((gpio_num_t)SENSOR_POWER_PIN);
  pinMode(SENSOR_POWER_PIN, OUTPUT);
  digitalWrite(SENSOR_POWER_PIN, on ? HIGH : LOW);
}

bool measure(Sample& s) {
  sensorPower(true);
  delay(10);  // module LDO start-up plus the SHT4x's own 1 ms

  Wire.begin(SDA, SCL);  // D4 = GPIO6, D5 = GPIO7
  bool ok = sht4x::read(s.temp, s.rh);
  Wire.end();
  // Release the bus before cutting power, or the pins back-feed the module
  // through its ESD diodes.
  pinMode(SDA, INPUT);
  pinMode(SCL, INPUT);
  sensorPower(false);

  s.vbat = 0;
  if (HAS_VBAT_DIVIDER) {
    analogReadMilliVolts(VBAT_PIN);  // first sample after wake is unreliable
    s.vbat = analogReadMilliVolts(VBAT_PIN) * 2 / 1000.0f * VBAT_SCALE;
  }
  s.takenAt = nowS();
  return ok;
}

void remember(const Sample& s) {
  if (buffered == BUFFER_SIZE) {  // drop the oldest
    memmove(buffer, buffer + 1, sizeof(Sample) * (BUFFER_SIZE - 1));
    buffered--;
  }
  buffer[buffered++] = s;
}

bool connectWifi() {
  WiFi.persistent(false);  // credentials come from secrets.h, keep flash out of it
  WiFi.mode(WIFI_STA);

  static unsigned long associatedAt = 0;
  static bool hooked = false;
  if (!hooked) {
    WiFi.onEvent([](WiFiEvent_t, WiFiEventInfo_t) { associatedAt = millis(); },
                 ARDUINO_EVENT_WIFI_STA_CONNECTED);
    hooked = true;
  }
  associatedAt = 0;

  bool targeted = apChannel > 0 && failures < FULL_SCAN_AFTER_FAILURES;
  bool reuseLease = targeted && leaseIp != 0 && nowS() - leaseAt < LEASE_REUSE_S;
  if (reuseLease) {
    WiFi.config(IPAddress(leaseIp), IPAddress(leaseGw), IPAddress(leaseMask), IPAddress(leaseDns));
  }
  if (targeted) {
    WiFi.begin(WIFI_SSID, WIFI_PASS, apChannel, apBssid);
  } else {
    WiFi.begin(WIFI_SSID, WIFI_PASS);
  }

  const char* mode = !targeted ? "scan" : reuseLease ? "targeted+lease" : "targeted";
  unsigned long start = millis();
  while (WiFi.status() != WL_CONNECTED) {
    if (millis() - start > WIFI_TIMEOUT_MS) {
      logf("wifi: timeout (%s), status %d\n", mode, WiFi.status());
      leaseIp = 0;  // the lease may be what failed; ask DHCP next time
      return false;
    }
    delay(10);
  }

  memcpy(apBssid, WiFi.BSSID(), 6);
  apChannel = WiFi.channel();
  if (!reuseLease) {
    leaseIp = WiFi.localIP();
    leaseGw = WiFi.gatewayIP();
    leaseMask = WiFi.subnetMask();
    leaseDns = WiFi.dnsIP(0);
    leaseAt = nowS();
  }
  logf("wifi: %s connect in %lu ms (associated at %lu ms), ip %s, rssi %d dBm\n",
                mode, millis() - start, associatedAt ? associatedAt - start : 0UL,
                WiFi.localIP().toString().c_str(), WiFi.RSSI());
  return true;
}

void appendSample(String& body, const Sample& s, time_t now, bool current) {
  char item[128];
  int n = snprintf(item, sizeof(item), "{\"rh\":%.2f,\"temp\":%.2f,\"age_s\":%ld",
                   s.rh, s.temp, (long)(now - s.takenAt));
  if (HAS_VBAT_DIVIDER) n += snprintf(item + n, sizeof(item) - n, ",\"vbat\":%.3f", s.vbat);
  if (current) {
    n += snprintf(item + n, sizeof(item) - n, ",\"rssi\":%d,\"fw\":\"%s\"", WiFi.RSSI(), FW_VERSION);
  }
  snprintf(item + n, sizeof(item) - n, "}");

  if (body.length() > 1) body += ",";
  body += item;
}

// Sends the buffered readings plus the current one, oldest first.
bool post(const Sample* current) {
  time_t now = nowS();
  String body = "[";
  for (int i = 0; i < buffered; i++) appendSample(body, buffer[i], now, false);
  if (current) appendSample(body, *current, now, true);
  body += "]";

  // https:// in HUMI_URL verifies the server against ca.h; http:// is for the
  // bench server on the LAN.
  WiFiClient plain;
  WiFiClientSecure tls;
  bool secure = strncmp(HUMI_URL, "https://", 8) == 0;
  if (secure) tls.setCACert(ROOT_CA);

  HTTPClient http;
  http.setTimeout(5000);
  if (secure) {
    http.begin(tls, HUMI_URL);
  } else {
    http.begin(plain, HUMI_URL);
  }
  http.addHeader("Content-Type", "application/json");
  http.addHeader("Authorization", "Bearer " HUMI_TOKEN);

  int code = http.POST(body);
  String reply = code > 0 ? http.getString() : http.errorToString(code);
  http.end();

  logf("post: %s -> %d %s\n", body.c_str(), code, reply.c_str());
  if (code != 200) return false;

  // The server owns the cadence; pick up interval_s without a JSON library.
  int at = reply.indexOf("\"interval_s\":");
  if (at >= 0) {
    int s = reply.substring(at + 13).toInt();
    if (s >= 10 && s <= 86400) intervalS = s;
  }
  return true;
}

unsigned long cycleStart = 0;  // millis() when this reading began

// Ends a cycle. On battery it deep-sleeps and never returns. With a USB host
// attached it waits awake instead, so the port stays up for flashing and logs.
void finishCycle() {
  WiFi.disconnect(true);
  WiFi.mode(WIFI_OFF);

  // Keep a fixed cadence: subtract the time already spent awake.
  uint64_t intervalUs = (uint64_t)intervalS * 1000000ULL;
  uint64_t awakeUs = (uint64_t)(millis() - cycleStart) * 1000ULL;
  uint64_t sleepUs = awakeUs < intervalUs ? intervalUs - awakeUs : 1000000ULL;
  bool usbHost = HWCDC::isPlugged();

  logf("%s: %llu s, awake %lu ms, buffered %d, failures %d\n\n",
       usbHost ? "wait (usb)" : "sleep", sleepUs / 1000000ULL,
       (unsigned long)(awakeUs / 1000), buffered, failures);
  flushLog();
  logBuf = "";

  if (usbHost) {
    delay(sleepUs / 1000);
    return;
  }

  if (SENSOR_POWER_PIN >= 0) {
    digitalWrite(SENSOR_POWER_PIN, LOW);
    gpio_hold_en((gpio_num_t)SENSOR_POWER_PIN);  // stay low through deep sleep
    gpio_deep_sleep_hold_en();
  }
  esp_sleep_enable_timer_wakeup(sleepUs);
  esp_deep_sleep_start();
}

void readAndSend() {
  Sample s;
  bool measured = measure(s);
  if (measured) {
    logf("sht4x: %.1f %%RH  %.2f C\n", s.rh, s.temp);
  } else {
    logf("sht4x: read failed\n");
  }
  if (HAS_VBAT_DIVIDER) logf("vbat: %.3f V\n", s.vbat);

  if (!measured && buffered == 0) return;  // nothing to send

  bool sent = false;
  if (connectWifi()) {
    const Sample* current = measured ? &s : nullptr;
    // Two attempts, then give up until the next wake (build note 4).
    sent = post(current) || post(current);
  }

  if (sent) {
    buffered = 0;
    failures = 0;
  } else {
    failures++;
    if (measured) remember(s);
  }
}

void setup() {
  Serial.begin(115200);
  Serial.setTxTimeoutMs(0);  // on battery there is no USB host; never block on it
}

// On battery every wake is a fresh boot that runs loop() once and sleeps.
// Only on USB does loop() actually repeat.
void loop() {
  static bool firstCycle = true;
  const char* cause = !firstCycle ? "usb"
                      : esp_sleep_get_wakeup_cause() == ESP_SLEEP_WAKEUP_TIMER ? "timer"
                      : "power-on";
  cycleStart = firstCycle ? 0 : millis();  // the first cycle counts from boot
  firstCycle = false;

  logf("humi node %s, wake %s\n", FW_VERSION, cause);
  readAndSend();
  finishCycle();
}
