package provider

// Acceptance tests: real Terraform or OpenTofu runs (plan, apply, refresh,
// import, destroy) of the provider against the in-memory fake Robot API in
// acc_fake_test.go. No Hetzner account is involved.
//
// They run only with TF_ACC set (`task testacc`). terraform-plugin-testing
// downloads the latest Terraform unless TF_ACC_TERRAFORM_PATH names a binary;
// point it at `tofu` to run the same tests on OpenTofu.

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

var testAccProviders = map[string]func() (tfprotov6.ProviderServer, error){
	"hrobot": providerserver.NewProtocol6WithError(New("test")()),
}

// replaced asserts that the plan destroys and re-creates the resource.
func replaced(addr string) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
		plancheck.ExpectResourceAction(addr, plancheck.ResourceActionDestroyBeforeCreate),
	}}
}

// identityMatches asserts that the resource identity is set and that each
// identity attribute equals the state attribute of the same name.
func identityMatches(addr string, attrs ...string) []statecheck.StateCheck {
	checks := make([]statecheck.StateCheck, len(attrs))
	for i, a := range attrs {
		checks[i] = statecheck.ExpectIdentityValueMatchesState(addr, tfjsonpath.New(a))
	}
	return checks
}

// updatedInPlace asserts that an import plan updates the resource in place.
func updatedInPlace(addr string) resource.ImportPlanChecks {
	return resource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{
		plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
	}}
}

// importByIdentity imports the resource with an `import { identity = ... }`
// block built from its stored identity. The import plan must be a no-op, so
// this also proves the imported state matches the configuration.
func importByIdentity(addr string) resource.TestStep {
	return resource.TestStep{
		ResourceName:    addr,
		ImportState:     true,
		ImportStateKind: resource.ImportBlockWithResourceIdentity,
	}
}

// checkFake runs fn against the fake's state under its lock.
func checkFake(f *fakeRobot, fn func() error) resource.TestCheckFunc {
	return func(*terraform.State) error {
		var err error
		f.with(func() { err = fn() })
		return err
	}
}

func TestAccSSHKey(t *testing.T) {
	f := newFakeRobot(t)
	cfg := func(name string) string {
		// The trailing newline is what `file("id.pub")` yields; the API drops it.
		return f.providerConfig() + fmt.Sprintf(`
resource "hrobot_ssh_key" "test" {
  name       = %q
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest deploy@example\n"
}
`, name)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy: checkFake(f, func() error {
			if len(f.keys) != 0 {
				return fmt.Errorf("%d key(s) left after destroy", len(f.keys))
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				Config:            cfg("deploy"),
				ConfigStateChecks: identityMatches("hrobot_ssh_key.test", "fingerprint"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("hrobot_ssh_key.test", "name", "deploy"),
					resource.TestCheckResourceAttrPair("hrobot_ssh_key.test", "id", "hrobot_ssh_key.test", "fingerprint"),
				),
			},
			{
				Config: cfg("renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("hrobot_ssh_key.test", plancheck.ResourceActionUpdate),
				}},
				Check: resource.TestCheckResourceAttr("hrobot_ssh_key.test", "name", "renamed"),
			},
			{
				ResourceName:      "hrobot_ssh_key.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Import reads the key as the API stores it, without the newline.
				ImportStateVerifyIgnore: []string{"public_key"},
			},
			{
				// The API stores the key without the newline `file()` adds:
				// importing plans an in-place update, never a replacement.
				ResourceName:       "hrobot_ssh_key.test",
				ImportState:        true,
				ImportStateKind:    resource.ImportBlockWithResourceIdentity,
				ExpectNonEmptyPlan: true,
				ImportPlanChecks:   updatedInPlace("hrobot_ssh_key.test"),
			},
		},
	})
}

func TestAccRDNS(t *testing.T) {
	f := newFakeRobot(t)
	cfg := func(ptr string) string {
		return f.providerConfig() + fmt.Sprintf(`
resource "hrobot_rdns" "test" {
  ip  = "2001:db8::10"
  ptr = %q
}
`, ptr)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy: checkFake(f, func() error {
			if len(f.rdns) != 0 {
				return fmt.Errorf("rdns entries left after destroy: %v", f.rdns)
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				Config:            cfg("host.example.com"),
				ConfigStateChecks: identityMatches("hrobot_rdns.test", "ip"),
				Check:             resource.TestCheckResourceAttr("hrobot_rdns.test", "ptr", "host.example.com"),
			},
			{Config: cfg("other.example.com"), Check: resource.TestCheckResourceAttr("hrobot_rdns.test", "ptr", "other.example.com")},
			{ResourceName: "hrobot_rdns.test", ImportState: true, ImportStateVerify: true},
			importByIdentity("hrobot_rdns.test"),
		},
	})
}

func TestAccFailoverIP(t *testing.T) {
	f := newFakeRobot(t)
	f.addFailover("198.51.100.10", 1001, "192.0.2.1")
	cfg := func(active string) string {
		return f.providerConfig() + fmt.Sprintf(`
resource "hrobot_failover_ip" "test" {
  ip               = "198.51.100.10"
  active_server_ip = %q
}
`, active)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy: checkFake(f, func() error {
			if a := f.failovers["198.51.100.10"].Active; a != nil {
				return fmt.Errorf("failover IP still routed to %s after destroy", *a)
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				Config:            cfg("192.0.2.1"),
				ConfigStateChecks: identityMatches("hrobot_failover_ip.test", "ip"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("hrobot_failover_ip.test", "active_server_ip", "192.0.2.1"),
					resource.TestCheckResourceAttr("hrobot_failover_ip.test", "server_number", "1001"),
				),
			},
			{Config: cfg("192.0.2.2"), Check: resource.TestCheckResourceAttr("hrobot_failover_ip.test", "active_server_ip", "192.0.2.2")},
			{ResourceName: "hrobot_failover_ip.test", ImportState: true, ImportStateVerify: true},
			importByIdentity("hrobot_failover_ip.test"),
		},
	})
}

func TestAccFirewall(t *testing.T) {
	f := newFakeRobot(t)
	f.addServer(1001)
	f.addServer(1002)
	cfg := func(server int) string {
		return f.providerConfig() + fmt.Sprintf(`
resource "hrobot_firewall" "test" {
  server_number = %d
  status        = "active"

  input_rule = [{
    name       = "ssh"
    ip_version = "ipv4"
    action     = "accept"
    protocol   = "tcp"
    dst_port   = "22"
  }]

  # No ip_version: the API stores this as an ipv4 and an ipv6 rule.
  output_rule = [{
    name   = "all"
    action = "accept"
  }]
}
`, server)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:            cfg(1001),
				ConfigStateChecks: identityMatches("hrobot_firewall.test", "server_number"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("hrobot_firewall.test", "id", "1001"),
					resource.TestCheckResourceAttr("hrobot_firewall.test", "output_rule.#", "1"),
					resource.TestCheckResourceAttr("hrobot_firewall.test", "port", "main"),
				),
			},
			{
				// Moving to another server replaces the resource and resets
				// the old server's firewall.
				Config:           cfg(1002),
				ConfigPlanChecks: replaced("hrobot_firewall.test"),
				Check: checkFake(f, func() error {
					if old := f.firewalls[1001]; old.Status != "disabled" || len(old.Input) != 0 {
						return fmt.Errorf("server 1001 firewall not reset: %+v", old)
					}
					if f.firewalls[1002].Status != "active" {
						return fmt.Errorf("server 1002 firewall not active")
					}
					return nil
				}),
			},
			{
				ResourceName:      "hrobot_firewall.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Without prior state, import sees the expanded output rules.
				ImportStateVerifyIgnore: []string{"output_rule"},
			},
			{
				// Import sees the expanded output rules, so it plans an
				// in-place update that re-applies the equivalent rules.
				ResourceName:       "hrobot_firewall.test",
				ImportState:        true,
				ImportStateKind:    resource.ImportBlockWithResourceIdentity,
				ExpectNonEmptyPlan: true,
				ImportPlanChecks:   updatedInPlace("hrobot_firewall.test"),
			},
			{
				// The server is cancelled: the firewall drops out of state.
				PreConfig:          func() { f.removeServer(1002) },
				Config:             cfg(1002),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccFirewallValidation(t *testing.T) {
	f := newFakeRobot(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{{
			Config: f.providerConfig() + `
resource "hrobot_firewall" "test" {
  server_number = 1001
  status        = "on"
  input_rule    = [{ action = "allow" }]
}
`,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`(?s)value must be one of.*value must be one of`),
		}},
	})
}

func vswitchConfig(f *fakeRobot, name string, vlan int, servers string) string {
	return f.providerConfig() + fmt.Sprintf(`
resource "hrobot_vswitch" "test" {
  name           = %q
  vlan           = %d
  server_numbers = %s
}
`, name, vlan, servers)
}

func TestAccVSwitch(t *testing.T) {
	f := newFakeRobot(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy: checkFake(f, func() error {
			if len(f.vswitches) != 0 {
				return fmt.Errorf("%d vSwitch(es) left after destroy", len(f.vswitches))
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				// Invalid config is rejected at plan time (first, because
				// the post-test destroy reuses the last step's config).
				Config:      vswitchConfig(f, "backend-2", 100, "[]"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must be between 4000 and 4091`),
			},
			{
				Config:            vswitchConfig(f, "backend", 4010, "[1001, 1002]"),
				ConfigStateChecks: identityMatches("hrobot_vswitch.test", "id"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("hrobot_vswitch.test", "server_numbers.#", "2"),
					resource.TestCheckResourceAttr("hrobot_vswitch.test", "cancellation_date", "now"),
				),
			},
			{
				Config: vswitchConfig(f, "backend-2", 4010, "[1002, 1003]"),
				Check: checkFake(f, func() error {
					for _, vs := range f.vswitches {
						if vs.Name != "backend-2" || len(vs.Servers) != 2 || !vs.Servers[1003] || vs.Servers[1001] {
							return fmt.Errorf("vSwitch not updated: %+v", vs)
						}
					}
					return nil
				}),
			},
			{
				ResourceName:      "hrobot_vswitch.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			importByIdentity("hrobot_vswitch.test"),
		},
	})
}

func TestAccVSwitchFailedCreateIsTainted(t *testing.T) {
	f := newFakeRobot(t)
	cfg := vswitchConfig(f, "backend", 4010, "[1001]")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				// The vSwitch is created, then attaching the server fails.
				PreConfig:   func() { f.failNext("POST", fmt.Sprintf("/vswitch/%d/server", f.nextVSwID+1)) },
				Config:      cfg,
				ExpectError: regexp.MustCompile(`add servers failed`),
			},
			{
				// It was saved as tainted, so it is replaced, not duplicated.
				Config:           cfg,
				ConfigPlanChecks: replaced("hrobot_vswitch.test"),
				Check: checkFake(f, func() error {
					if len(f.vswitches) != 1 {
						return fmt.Errorf("want 1 vSwitch, have %d", len(f.vswitches))
					}
					return nil
				}),
			},
		},
	})
}

func subAccountConfig(f *fakeRobot, comment string) string {
	return f.providerConfig() + `
resource "hrobot_storagebox_subaccount" "test" {
  storagebox_id  = 4242
  home_directory = "/backups"
  ssh            = true
` + comment + `
}
`
}

func TestAccStorageBoxSubAccount(t *testing.T) {
	f := newFakeRobot(t)
	f.addBox(4242)
	addr := "hrobot_storagebox_subaccount.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy: checkFake(f, func() error {
			if n := len(f.boxes[4242].Subs); n != 0 {
				return fmt.Errorf("%d sub-account(s) left after destroy", n)
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				Config:            subAccountConfig(f, `  comment = "nightly"`),
				ConfigStateChecks: identityMatches(addr, "storagebox_id", "username"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "id", "4242/u4242-sub1"),
					resource.TestCheckResourceAttr(addr, "password", "generated-secret"),
					resource.TestCheckResourceAttr(addr, "ssh", "true"),
					resource.TestCheckResourceAttr(addr, "samba", "false"),
				),
			},
			{
				ResourceName:            addr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			importByIdentity(addr),
			{
				// Removing the comment clears it.
				Config: subAccountConfig(f, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(addr, "comment"),
					resource.TestCheckResourceAttr(addr, "password", "generated-secret"),
					checkFake(f, func() error {
						if c := f.boxes[4242].Subs[0]["comment"]; c != "" {
							return fmt.Errorf("comment not cleared: %q", c)
						}
						return nil
					}),
				),
			},
			{
				Config: subAccountConfig(f, `  comment = ""`),
				Check:  resource.TestCheckResourceAttr(addr, "comment", ""),
			},
		},
	})
}

func TestAccStorageBoxSubAccountFailedCreateIsTainted(t *testing.T) {
	f := newFakeRobot(t)
	f.addBox(4242)
	addr := "hrobot_storagebox_subaccount.test"
	cfg := subAccountConfig(f, "")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Created, then the re-read fails.
				PreConfig:   func() { f.failNext("GET", "/storagebox/4242/subaccount") },
				Config:      cfg,
				ExpectError: regexp.MustCompile(`re-read failed`),
			},
			{
				// The tainted sub-account kept its one-time password.
				RefreshState:       true,
				ExpectNonEmptyPlan: true, // tainted: the next plan replaces it
				Check:              resource.TestCheckResourceAttr(addr, "password", "generated-secret"),
			},
			{
				Config:           cfg,
				ConfigPlanChecks: replaced(addr),
				Check: checkFake(f, func() error {
					if n := len(f.boxes[4242].Subs); n != 1 {
						return fmt.Errorf("want 1 sub-account, have %d", n)
					}
					return nil
				}),
			},
		},
	})
}

func snapshotConfig(f *fakeRobot, comment string) string {
	return f.providerConfig() + fmt.Sprintf(`
resource "hrobot_storagebox_snapshot" "test" {
  storagebox_id = 4242
  comment       = %q
}
`, comment)
}

func TestAccStorageBoxSnapshot(t *testing.T) {
	f := newFakeRobot(t)
	f.addBox(4242)
	addr := "hrobot_storagebox_snapshot.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy: checkFake(f, func() error {
			if n := len(f.boxes[4242].Snapshots); n != 0 {
				return fmt.Errorf("%d snapshot(s) left after destroy", n)
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				Config:            snapshotConfig(f, "before upgrade"),
				ConfigStateChecks: identityMatches(addr, "storagebox_id", "name"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "comment", "before upgrade"),
					resource.TestCheckResourceAttr(addr, "automatic", "false"),
				),
			},
			{ResourceName: addr, ImportState: true, ImportStateVerify: true},
			importByIdentity(addr),
			{Config: snapshotConfig(f, ""), Check: resource.TestCheckResourceAttr(addr, "comment", "")},
		},
	})
}

func TestAccStorageBoxSnapshotFailedCreateIsTainted(t *testing.T) {
	f := newFakeRobot(t)
	f.addBox(4242)
	cfg := snapshotConfig(f, "before upgrade")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				PreConfig:   func() { f.failNext("POST", "/storagebox/4242/snapshot/2026-11-28T10-00-01/comment") },
				Config:      cfg,
				ExpectError: regexp.MustCompile(`snapshot comment failed`),
			},
			{
				Config:           cfg,
				ConfigPlanChecks: replaced("hrobot_storagebox_snapshot.test"),
				Check: checkFake(f, func() error {
					if n := len(f.boxes[4242].Snapshots); n != 1 {
						return fmt.Errorf("want 1 snapshot, have %d", n)
					}
					return nil
				}),
			},
		},
	})
}

func TestAccStorageBoxSnapshotPlan(t *testing.T) {
	f := newFakeRobot(t)
	f.addBox(4343)
	addr := "hrobot_storagebox_snapshot_plan.test"
	cfg := func(schedule string) string {
		return f.providerConfig() + `
resource "hrobot_storagebox_snapshot_plan" "test" {
  storagebox_id = 4343
  max_snapshots = 7
` + schedule + `
}
`
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy: checkFake(f, func() error {
			if s := f.boxes[4343].Plan["status"]; s != "disabled" {
				return fmt.Errorf("snapshot plan status %v after destroy, want disabled", s)
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				// Rejected at plan time (first: the post-test destroy reuses
				// the last step's config).
				Config:      cfg("  hour = 24"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must be between 0 and 23`),
			},
			{
				Config:            cfg("  minute = 0\n  hour = 3"),
				ConfigStateChecks: identityMatches(addr, "storagebox_id"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "status", "enabled"),
					resource.TestCheckResourceAttr(addr, "hour", "3"),
				),
			},
			{
				Config: cfg("  hour = 4"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "hour", "4"),
					resource.TestCheckNoResourceAttr(addr, "minute"),
				),
			},
			{ResourceName: addr, ImportState: true, ImportStateVerify: true},
			importByIdentity(addr),
		},
	})
}
