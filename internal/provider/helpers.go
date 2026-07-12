package provider

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaltenecker-kg/hrobot-go"
)

// isNotFound reports whether err is an hrobot API error with HTTP status 404,
// so Read can drop a resource that was deleted outside of Terraform.
func isNotFound(err error) bool {
	var e *hrobot.Error
	return errors.As(err, &e) && e.Status == http.StatusNotFound
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
