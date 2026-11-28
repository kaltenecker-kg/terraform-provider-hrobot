package provider

import (
	"context"
	"testing"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

func TestSetSSHKeyResourceModelKeepsConfiguredWhitespace(t *testing.T) {
	// `public_key = file("id.pub")` carries a trailing newline the API drops.
	m := sshKeyResourceModel{PublicKey: types.StringValue("ssh-ed25519 AAAA deploy@example\n")}
	setSSHKeyResourceModel(&m, &hrobot.SSHKey{Fingerprint: "aa:bb", Data: "ssh-ed25519 AAAA deploy@example"})
	if got := m.PublicKey.ValueString(); got != "ssh-ed25519 AAAA deploy@example\n" {
		t.Errorf("public_key = %q, want the configured value kept", got)
	}

	// Different key material is real drift and must come from the API.
	m = sshKeyResourceModel{PublicKey: types.StringValue("ssh-ed25519 OLD\n")}
	setSSHKeyResourceModel(&m, &hrobot.SSHKey{Fingerprint: "aa:bb", Data: "ssh-ed25519 NEW\n"})
	if got := m.PublicKey.ValueString(); got != "ssh-ed25519 NEW" {
		t.Errorf("public_key = %q, want the API value", got)
	}
}

func TestOptStringKeepEmpty(t *testing.T) {
	cases := []struct {
		name  string
		prior types.String
		api   string
		want  types.String
	}{
		{"configured empty stays empty", types.StringValue(""), "", types.StringValue("")},
		{"null stays null", types.StringNull(), "", types.StringNull()},
		{"api value wins", types.StringValue(""), "set elsewhere", types.StringValue("set elsewhere")},
		{"unset prior with api value", types.StringNull(), "x", types.StringValue("x")},
	}
	for _, c := range cases {
		if got := optStringKeepEmpty(c.prior, c.api); !got.Equal(c.want) {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

func TestSubAccountInputAlwaysSendsComment(t *testing.T) {
	// A removed comment must be sent as "" so the API clears it.
	in := subAccountInput(&storageBoxSubAccountResourceModel{Comment: types.StringNull()})
	if in.Comment == nil || *in.Comment != "" {
		t.Errorf("comment = %v, want pointer to empty string", in.Comment)
	}
}

func TestIsNotFoundRecognisesResourceCodes(t *testing.T) {
	if !isNotFound(hrobot.NewAPIError(hrobot.ErrServerNotFound, "server not found")) {
		t.Error("SERVER_NOT_FOUND without a status should count as not found")
	}
	if isNotFound(hrobot.NewAPIError(hrobot.ErrRateLimitExceeded, "slow down")) {
		t.Error("rate limit is not a not-found error")
	}
}

func TestFirewallServerNumberRequiresReplace(t *testing.T) {
	s := rsSchema(t, &firewallResource{})
	attr, ok := s.Attributes["server_number"].(rschema.Int64Attribute)
	if !ok {
		t.Fatal("server_number is not an Int64Attribute")
	}
	var resp planmodifier.Int64Response
	resp.PlanValue = types.Int64Value(2)
	for _, m := range attr.PlanModifiers {
		m.PlanModifyInt64(context.Background(), planmodifier.Int64Request{
			State:       tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})},
			StateValue:  types.Int64Value(1),
			PlanValue:   types.Int64Value(2),
			ConfigValue: types.Int64Value(2),
			Plan:        tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})},
		}, &resp)
	}
	if !resp.RequiresReplace {
		t.Error("changing server_number must replace the firewall")
	}
}
