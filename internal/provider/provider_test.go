package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// TestProvider_Schema builds the full provider schema through the protocol
// server, which validates every resource and data-source schema, and asserts
// the expected type names are registered.
func TestProvider_Schema(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New("test")())()

	resp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema returned error: %v", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("schema diagnostic: %s: %s", d.Summary, d.Detail)
		}
	}

	wantData := []string{
		"hrobot_server", "hrobot_servers",
		"hrobot_ssh_key", "hrobot_ssh_keys",
		"hrobot_rdns",
		"hrobot_failover", "hrobot_failovers",
		"hrobot_vswitch", "hrobot_vswitches",
		"hrobot_ip", "hrobot_ips",
		"hrobot_subnet", "hrobot_subnets",
		"hrobot_storagebox", "hrobot_storageboxes",
		"hrobot_storagebox_subaccounts", "hrobot_storagebox_snapshots",
		"hrobot_boot", "hrobot_traffic",
	}
	for _, n := range wantData {
		if _, ok := resp.DataSourceSchemas[n]; !ok {
			t.Errorf("data source %q not registered", n)
		}
	}
	wantResources := []string{
		"hrobot_firewall",
		"hrobot_ssh_key",
		"hrobot_rdns",
		"hrobot_vswitch",
		"hrobot_failover_ip",
		"hrobot_storagebox_subaccount",
		"hrobot_storagebox_snapshot",
		"hrobot_storagebox_snapshot_plan",
	}
	for _, n := range wantResources {
		if _, ok := resp.ResourceSchemas[n]; !ok {
			t.Errorf("resource %q not registered", n)
		}
	}
}

func TestValidateBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"https", "https://robot-ws.your-server.de", false},
		{"https with port and path", "https://example.com:8443/api", false},
		{"http localhost", "http://localhost:8080", false},
		{"http 127.0.0.1", "http://127.0.0.1:8080", false},
		{"http ipv6 loopback", "http://[::1]:8080", false},
		{"http remote host", "http://robot-ws.your-server.de", true},
		{"http private ip", "http://192.168.1.10", true},
		{"no scheme", "robot-ws.your-server.de", true},
		{"unsupported scheme", "ftp://example.com", true},
		{"unparseable", "https://ex ample.com", true},
		{"https empty host", "https://", true},
		{"opaque https", "https:example.com", true},
		{"http empty host", "http://", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBaseURL(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBaseURL(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
		})
	}
}
