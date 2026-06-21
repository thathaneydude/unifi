// Package unifi provides authenticated access to the UniFi Network and Protect
// integration APIs, locally or remotely, plus a version-agnostic WebSocket
// layer for Protect real-time subscriptions.
//
// Construct a connection with Local or Remote, then reach the latest generated
// client via Conn.Network or Conn.Protect, or build a specific coexisting
// version from lib/<app>/<version> using the connection's base URL, request
// editor, and HTTP client.
package unifi
