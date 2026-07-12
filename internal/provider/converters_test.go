package provider

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

// dsSchema returns a data source's schema.
func dsSchema(t *testing.T, ds datasource.DataSource) dschema.Schema {
	t.Helper()
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %+v", resp.Diagnostics)
	}
	return resp.Schema
}

// mustSetState serializes model into a state built from schema, asserting that
// the model's tfsdk tags and value types match the schema (the check that would
// otherwise only fire at runtime during State.Set).
func mustSetState(t *testing.T, s dschema.Schema, model any) {
	t.Helper()
	st := tfsdk.State{Schema: s}
	diags := st.Set(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("state.Set diagnostics: %+v", diags)
	}
}

func TestSetSSHKeyModel(t *testing.T) {
	var m sshKeyModel
	setSSHKeyModel(&m, &hrobot.SSHKey{
		Name: "laptop", Fingerprint: "aa:bb", Type: "ED25519", Size: 256,
		Data: "ssh-ed25519 AAAA", CreatedAt: hrobot.BerlinTime{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
	})
	if m.Name.ValueString() != "laptop" || m.Size.ValueInt64() != 256 {
		t.Errorf("unexpected mapping: %+v", m)
	}
	mustSetState(t, dsSchema(t, &sshKeyDataSource{}), &m)
}

func TestSetFailoverModel(t *testing.T) {
	active := "5.6.7.8"
	var m failoverModel
	setFailoverModel(&m, &hrobot.Failover{IP: "1.2.3.4", Netmask: "255.255.255.255", ServerIP: "9.9.9.9", ServerNumber: 42, ActiveServerIP: &active})
	if m.ActiveServerIP.ValueString() != active {
		t.Errorf("active_server_ip = %q, want %q", m.ActiveServerIP.ValueString(), active)
	}
	mustSetState(t, dsSchema(t, &failoverDataSource{}), &m)

	var mNull failoverModel
	setFailoverModel(&mNull, &hrobot.Failover{IP: "1.2.3.4", ServerNumber: 1})
	if !mNull.ActiveServerIP.IsNull() {
		t.Errorf("expected null active_server_ip when API returns nil")
	}
	mustSetState(t, dsSchema(t, &failoverDataSource{}), &mNull)
}

func TestSetIPModel(t *testing.T) {
	var m ipModel
	setIPModel(&m, &hrobot.IPAddress{
		IP: net.ParseIP("1.2.3.4"), ServerIP: net.ParseIP("9.9.9.9"),
		Mask: 32, ServerNumber: 7, TrafficWarnings: true, TrafficMonthly: 1000,
	}) // Gateway and Broadcast are nil → null strings
	if !m.Gateway.IsNull() || !m.Broadcast.IsNull() {
		t.Errorf("expected nil net.IP to map to null string")
	}
	if m.IP.ValueString() != "1.2.3.4" {
		t.Errorf("ip = %q", m.IP.ValueString())
	}
	mustSetState(t, dsSchema(t, &ipDataSource{}), &m)
}

func TestSetSubnetModel(t *testing.T) {
	var m subnetModel
	setSubnetModel(&m, &hrobot.SubnetResource{IP: "2001:db8::", Mask: 64, Gateway: "2001:db8::1", ServerNumber: 3, Failover: true})
	mustSetState(t, dsSchema(t, &subnetDataSource{}), &m)
}

func TestSetVSwitchModel(t *testing.T) {
	var m vswitchModel
	setVSwitchModel(&m, &hrobot.VSwitch{
		ID: 1, Name: "vsw", VLAN: 4000,
		Servers:      []hrobot.VSwitchServer{{ServerIP: "1.2.3.4", ServerNumber: 5, Status: "ready"}},
		Subnets:      []hrobot.VSwitchSubnet{{IP: "10.0.0.0", Mask: 24, Gateway: "10.0.0.1"}},
		CloudNetwork: []hrobot.CloudNetwork{{ID: 2, IP: "10.1.0.0", Mask: 24, Gateway: "10.1.0.1"}},
	})
	if len(m.Servers) != 1 || m.Servers[0].ServerNumber.ValueInt64() != 5 {
		t.Errorf("server mapping wrong: %+v", m.Servers)
	}
	mustSetState(t, dsSchema(t, &vswitchDataSource{}), &m)
}

func TestSetStorageBoxModel(t *testing.T) {
	var m storageBoxModel
	setStorageBoxModel(&m, &hrobot.StorageBox{ID: 1, Login: "u1", Name: "box", Product: "BX11", DiskQuota: 1024, SSH: true})
	mustSetState(t, dsSchema(t, &storageBoxDataSource{}), &m)
}

func TestSetBootModel(t *testing.T) {
	var m bootModel
	// Only rescue + plesk present; linux/vnc/windows/cpanel nil → null objects.
	setBootModel(&m, &hrobot.BootConfig{
		Rescue: &hrobot.RescueConfig{Active: true},
		Plesk:  &hrobot.PleskConfig{Active: true, Hostname: "host.example"},
	})
	if m.Rescue == nil || !m.Rescue.Active.ValueBool() {
		t.Errorf("rescue not mapped")
	}
	if m.Linux != nil || m.CPanel != nil {
		t.Errorf("absent boot configs should stay nil")
	}
	mustSetState(t, dsSchema(t, &bootDataSource{}), &m)
}

func TestTrafficModel(t *testing.T) {
	ctx := context.Background()
	goMap := map[string]trafficStatsModel{
		"1.2.3.4": {In: types.Float64Value(1), Out: types.Float64Value(2), Sum: types.Float64Value(3)},
	}
	data, diags := types.MapValueFrom(ctx, types.ObjectType{AttrTypes: trafficStatsAttrTypes}, goMap)
	if diags.HasError() {
		t.Fatalf("MapValueFrom diagnostics: %+v", diags)
	}
	m := trafficModel{
		Type: types.StringValue("day"), From: types.StringValue("2026-07-01T00"),
		To: types.StringValue("2026-07-02T00"), IP: types.StringValue("1.2.3.4"), Data: data,
	}
	mustSetState(t, dsSchema(t, &trafficDataSource{}), &m)
}
