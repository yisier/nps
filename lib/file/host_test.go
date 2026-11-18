package file

import "testing"

func newTestHost(host string) *Host {
	return &Host{Host: host, Scheme: "all", Flow: &Flow{}}
}

// 覆盖 #340：同一域名允许存在多条记录，但只有停用的记录不参与查重，
// 保证同一时间只有一条记录生效。
func TestIsHostExistOnlyMatchesEnabledHosts(t *testing.T) {
	db := newTestDb(t)

	active := newTestHost("a.example.com")
	if err := db.NewHost(active); err != nil {
		t.Fatal(err)
	}

	dup := newTestHost("a.example.com")
	if !db.IsHostExist(dup) {
		t.Fatal("adding a duplicate of an enabled host should be rejected")
	}

	// 停用后不再阻塞同名记录
	active.IsClose = true
	if db.IsHostExist(dup) {
		t.Fatal("a disabled host should not block a duplicate record")
	}
	if err := db.NewHost(dup); err != nil {
		t.Fatalf("creating a record for a disabled host should succeed: %v", err)
	}

	// 两条同名记录同时启用时必须被拒绝
	active.IsClose = false
	if !db.IsHostExist(dup) {
		t.Fatal("expected a conflict when two records of the same domain are enabled")
	}
}

func TestIsHostExistIgnoresItself(t *testing.T) {
	db := newTestDb(t)

	h := newTestHost("b.example.com")
	if err := db.NewHost(h); err != nil {
		t.Fatal(err)
	}
	if db.IsHostExist(h) {
		t.Fatal("the host itself should not be treated as a conflict")
	}
}

func TestIsHostExistDistinguishesSchemeAndLocation(t *testing.T) {
	db := newTestDb(t)

	h := newTestHost("c.example.com")
	h.Scheme = "http"
	h.Location = "/"
	if err := db.NewHost(h); err != nil {
		t.Fatal(err)
	}

	same := newTestHost("c.example.com")
	same.Scheme = "http"
	if !db.IsHostExist(same) {
		t.Fatal("same scheme and default location should conflict")
	}

	otherLocation := newTestHost("c.example.com")
	otherLocation.Scheme = "http"
	otherLocation.Location = "/api"
	if db.IsHostExist(otherLocation) {
		t.Fatal("different locations should not conflict")
	}
}
