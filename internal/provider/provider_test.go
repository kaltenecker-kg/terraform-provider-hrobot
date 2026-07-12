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
	}
	for _, n := range wantResources {
		if _, ok := resp.ResourceSchemas[n]; !ok {
			t.Errorf("resource %q not registered", n)
		}
	}
}
