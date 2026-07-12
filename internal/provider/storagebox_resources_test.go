package provider

import (
	"testing"

	"github.com/kaltenecker-kg/hrobot-go"
)

func TestSetSubAccountResourceModel(t *testing.T) {
	var m storageBoxSubAccountResourceModel
	setSubAccountResourceModel(&m, 12345, &hrobot.StorageBoxSubAccount{
		Username: "u12345-sub1", AccountID: "A1", Server: "u12345.your-storagebox.de",
		HomeDirectory: "/backups", Samba: true, Comment: "nightly",
	})
	if m.ID.ValueString() != "12345/u12345-sub1" {
		t.Errorf("id = %q", m.ID.ValueString())
	}
	if !m.Password.IsNull() {
		t.Errorf("password should be untouched (null) by the setter")
	}
	mustSetResourceState(t, rsSchema(t, &storageBoxSubAccountResource{}), &m)
}

func TestSetSnapshotResourceModel(t *testing.T) {
	var m storageBoxSnapshotResourceModel
	setSnapshotResourceModel(&m, 12345, &hrobot.StorageBoxSnapshot{
		Name: "2026-07-12T20-00-00", Timestamp: "2026-07-12 20:00:00", Size: 1024, FilesystemSize: 2048,
	})
	if m.ID.ValueString() != "12345/2026-07-12T20-00-00" {
		t.Errorf("id = %q", m.ID.ValueString())
	}
	mustSetResourceState(t, rsSchema(t, &storageBoxSnapshotResource{}), &m)
}

func TestSetSnapshotPlanResourceModel(t *testing.T) {
	minute := 30
	var m storageBoxSnapshotPlanResourceModel
	setSnapshotPlanResourceModel(&m, 12345, &hrobot.StorageBoxSnapshotPlan{
		Status: "enabled", Minute: &minute, MaxSnapshots: 7, // Hour/DayOfWeek/... nil → null
	})
	if m.Minute.ValueInt64() != 30 || !m.Hour.IsNull() {
		t.Errorf("schedule mapping wrong: minute=%v hour-null=%v", m.Minute, m.Hour.IsNull())
	}
	if m.MaxSnapshots.ValueInt64() != 7 {
		t.Errorf("max_snapshots = %d", m.MaxSnapshots.ValueInt64())
	}
	mustSetResourceState(t, rsSchema(t, &storageBoxSnapshotPlanResource{}), &m)
}
