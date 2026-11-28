package provider

import (
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go/v2"
)

// isNotFound reports whether err means the object no longer exists, so Read
// can drop a resource that was deleted outside of Terraform. It defers to the
// library, which also recognises the per-resource NOT_FOUND codes.
func isNotFound(err error) bool {
	return hrobot.IsNotFoundError(err)
}

// optStringKeepEmpty maps an optional API string to state like optString, but
// keeps a configured empty string: the API reports "" and null identically, so
// returning null for a configured "" would make Terraform reject the apply as
// inconsistent with the plan.
func optStringKeepEmpty(prior types.String, api string) types.String {
	if api == "" && !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() == "" {
		return prior
	}
	return optString(api)
}

// strPtr returns a pointer to the string value, or nil when the value is null or
// unknown (so optional attributes are omitted from API requests).
func strPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

// boolPtr returns a pointer to the bool value, or nil when null/unknown.
func boolPtr(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

// intPtr returns a pointer to the int value, or nil when null/unknown.
func intPtr(v types.Int64) *int {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	n := int(v.ValueInt64())
	return &n
}

// int64FromPtr maps a *int (nullable API field) to a Terraform Int64.
func int64FromPtr(p *int) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}

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

// trafficWarningDeprecation is the deprecation notice shared by the
// traffic_warnings/traffic_hourly/traffic_daily/traffic_monthly attributes of
// the IP and subnet data sources.
const trafficWarningDeprecation = "Hetzner removes the traffic warning fields from the Robot API on 12 January 2027. " +
	"After that date this attribute is always false/0; it will be removed in a future major release."
