package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

func TestSetFirewallModel(t *testing.T) {
	var m firewallModel
	setFirewallModel(&m, &hrobot.FirewallConfig{
		ServerNumber: 321,
		Status:       hrobot.FirewallStatusActive,
		FilterIPv6:   true,
		WhitelistHOS: true,
		Port:         "main",
		Rules: hrobot.FirewallRules{
			Input: []hrobot.FirewallRule{{
				Name:      "ssh",
				IPVersion: hrobot.IPv4,
				Action:    hrobot.ActionAccept,
				Protocol:  hrobot.ProtocolTCP,
				DestPort:  "22",
			}},
			Output: []hrobot.FirewallRule{{
				Name:   "Allow all",
				Action: hrobot.ActionAccept,
			}},
		},
	})
	if m.ID.ValueString() != "321" || m.Port.ValueString() != "main" {
		t.Errorf("mapping wrong: id=%q port=%q", m.ID.ValueString(), m.Port.ValueString())
	}
	if len(m.InputRules) != 1 || len(m.OutputRules) != 1 {
		t.Fatalf("rules = %d/%d, want 1/1", len(m.InputRules), len(m.OutputRules))
	}
	// Unset API fields must map to null, not "".
	if !m.OutputRules[0].IPVersion.IsNull() || !m.OutputRules[0].Protocol.IsNull() {
		t.Errorf("unset rule fields should be null: %+v", m.OutputRules[0])
	}
	mustSetResourceState(t, rsSchema(t, &firewallResource{}), &m)
}

func TestFirewallRulesRoundTrip(t *testing.T) {
	in := []ruleModel{{
		Name:     types.StringValue("block mail"),
		Action:   types.StringValue("discard"),
		Protocol: types.StringValue("tcp"),
		DestPort: types.StringValue("25"),
		// unset attributes stay null
		IPVersion: types.StringNull(),
		SourceIP:  types.StringNull(),
		DestIP:    types.StringNull(),
	}}
	api := toAPIRules(in)
	if len(api) != 1 || api[0].IPVersion != "" || api[0].DestPort != "25" {
		t.Fatalf("toAPIRules = %+v", api)
	}
	back := fromAPIRules(api)
	if len(back) != 1 {
		t.Fatalf("fromAPIRules len = %d", len(back))
	}
	if !back[0].IPVersion.IsNull() {
		t.Errorf("empty ip_version should round-trip to null")
	}
	if back[0].DestPort.ValueString() != "25" || back[0].Action.ValueString() != "discard" {
		t.Errorf("round-trip lost fields: %+v", back[0])
	}
	if toAPIRules(nil) != nil || fromAPIRules(nil) != nil {
		t.Errorf("empty rule lists should map to nil")
	}
}

// rm builds a ruleModel the way config declares it (unset fields null).
func rm(name, ipVersion, action, protocol, dstPort string) ruleModel {
	opt := func(s string) types.String {
		if s == "" {
			return types.StringNull()
		}
		return types.StringValue(s)
	}
	return ruleModel{
		Name:      opt(name),
		IPVersion: opt(ipVersion),
		Action:    types.StringValue(action),
		Protocol:  opt(protocol),
		DestPort:  opt(dstPort),
	}
}

// ar builds the corresponding API rule.
func ar(name string, ipVersion hrobot.IPVersion, action hrobot.Action, protocol hrobot.Protocol, dstPort string) hrobot.FirewallRule {
	return hrobot.FirewallRule{
		Name:      name,
		IPVersion: ipVersion,
		Action:    action,
		Protocol:  protocol,
		DestPort:  dstPort,
	}
}

// TestReconcileRules_KeepsStateOnAPIExpansion reproduces the bug report:
// config declares 6 output rules, two of them version-less; the live config
// returns 8 because the API expanded the version-less rules into ipv4+ipv6
// entries grouped by version. Read must keep the state's compact form.
func TestReconcileRules_KeepsStateOnAPIExpansion(t *testing.T) {
	prior := firewallModel{
		InputRules: []ruleModel{
			rm("ssh", "ipv4", "accept", "tcp", "22"),
		},
		OutputRules: []ruleModel{
			rm("dns", "ipv4", "accept", "udp", "53"),
			rm("http", "ipv4", "accept", "tcp", "80"),
			rm("ntp", "ipv4", "accept", "udp", "123"),
			rm("block mail", "", "discard", "tcp", "25"),
			rm("block mail", "", "discard", "tcp", "465"),
			rm("Allow all", "", "accept", "", ""),
		},
	}
	live := hrobot.FirewallRules{
		Input: []hrobot.FirewallRule{
			ar("ssh", hrobot.IPv4, hrobot.ActionAccept, hrobot.ProtocolTCP, "22"),
		},
		// version-specific rules first, then the version-less ones expanded
		// and grouped per version (2x ipv4 + 2x ipv6), then the catch-all.
		Output: []hrobot.FirewallRule{
			ar("dns", hrobot.IPv4, hrobot.ActionAccept, hrobot.ProtocolUDP, "53"),
			ar("http", hrobot.IPv4, hrobot.ActionAccept, hrobot.ProtocolTCP, "80"),
			ar("ntp", hrobot.IPv4, hrobot.ActionAccept, hrobot.ProtocolUDP, "123"),
			ar("block mail", hrobot.IPv4, hrobot.ActionDiscard, hrobot.ProtocolTCP, "25"),
			ar("block mail", hrobot.IPv4, hrobot.ActionDiscard, hrobot.ProtocolTCP, "465"),
			ar("block mail", hrobot.IPv6, hrobot.ActionDiscard, hrobot.ProtocolTCP, "25"),
			ar("block mail", hrobot.IPv6, hrobot.ActionDiscard, hrobot.ProtocolTCP, "465"),
			ar("Allow all", "", hrobot.ActionAccept, "", ""),
		},
	}

	input, output := reconcileRules(&prior, live)
	if len(input) != 1 || len(output) != 6 {
		t.Fatalf("reconciled rules = %d/%d, want 1/6 (state kept)", len(input), len(output))
	}
	if !output[3].IPVersion.IsNull() {
		t.Errorf("state's version-less rule should be kept as-is")
	}
}

func TestReconcileRules_TakesLiveOnRealDrift(t *testing.T) {
	prior := firewallModel{
		OutputRules: []ruleModel{rm("Allow all", "", "accept", "", "")},
	}
	live := hrobot.FirewallRules{
		Output: []hrobot.FirewallRule{
			ar("block mail", "", hrobot.ActionDiscard, hrobot.ProtocolTCP, "25"),
			ar("Allow all", "", hrobot.ActionAccept, "", ""),
		},
	}
	input, output := reconcileRules(&prior, live)
	if input != nil {
		t.Errorf("input = %+v, want nil", input)
	}
	if len(output) != 2 || output[0].Name.ValueString() != "block mail" {
		t.Errorf("live rules should win on real drift: %+v", output)
	}
}

func TestReconcileRules_EmptyStateTakesLive(t *testing.T) {
	// Import path: state has no rules yet, live config must be adopted.
	prior := firewallModel{}
	live := hrobot.FirewallRules{
		Input: []hrobot.FirewallRule{
			ar("ssh", hrobot.IPv4, hrobot.ActionAccept, hrobot.ProtocolTCP, "22"),
		},
	}
	input, output := reconcileRules(&prior, live)
	if len(input) != 1 || output != nil {
		t.Errorf("reconciled rules = %d/%v, want live input adopted", len(input), output)
	}
}
