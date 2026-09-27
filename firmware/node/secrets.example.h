// Copy to secrets.h (gitignored) and fill in. One file per node: the token is
// what the server knows the node by.
#pragma once

#define WIFI_SSID  "your-network"
#define WIFI_PASS  "your-password"

// Bench: plain HTTP to the machine running the server on the LAN.
// Deployed: https://humi.example.com/api/v1/readings
#define HUMI_URL   "http://192.168.1.144:9820/api/v1/readings"

// From: nodectl -slug <slug> -name <Name>
#define HUMI_TOKEN "paste-the-token-here"
