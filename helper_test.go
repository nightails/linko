package main

import (
	"testing"
)

func TestRedactIP(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "IPv4 with port",
			input:    "192.168.1.1:8080",
			expected: "192.168.1.x",
		},
		{
			name:     "IPv6 with port",
			input:    "[2001:db8::1]:80",
			expected: "2001:db8::x",
		},
		{
			name:     "Hostname with port",
			input:    "example.com:443",
			expected: "example.com",
		},
		{
			name:     "IPv4 without port",
			input:    "127.0.0.1",
			expected: "127.0.0.1", // net.SplitHostPort fails, returns original
		},
		{
			name:     "Invalid IP with port",
			input:    "999.999.999.999:80",
			expected: "999.999.999.999", // ParseIP fails, returns host
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := redactIP(tc.input)
			if got != tc.expected {
				t.Errorf("redactIP(%q) = %q; want %q", tc.input, got, tc.expected)
			}
		})
	}
}
