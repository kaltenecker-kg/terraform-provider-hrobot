package provider

import (
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

// ipToString renders a net.IP as a Terraform string, mapping the empty/nil IP to
// a null value (rather than the literal "<nil>").
func ipToString(ip net.IP) types.String {
	if len(ip) == 0 {
		return types.StringNull()
	}
	return types.StringValue(ip.String())
}

// configureClient type-asserts the provider data into a *hrobot.Client, adding a
// diagnostic on mismatch. It returns nil during the provider's early Configure
// pass (when ProviderData is nil), matching the plugin-framework contract.
func configureClient(providerData any, diags *diag.Diagnostics) *hrobot.Client {
	if providerData == nil {
		return nil
	}
	c, ok := providerData.(*hrobot.Client)
	if !ok {
		diags.AddError("unexpected provider data type", fmt.Sprintf("got %T", providerData))
		return nil
	}
	return c
}
