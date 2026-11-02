package file

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestDb(t *testing.T) *DbUtils {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "conf"), 0755); err != nil {
		t.Fatal(err)
	}
	return &DbUtils{JsonDb: NewJsonDb(dir)}
}

func newTestClient() *Client {
	return &Client{
		Status: true,
		Cnf:    &Config{},
		Flow:   &Flow{},
	}
}

func TestNewClientIdsOnlyMoveForward(t *testing.T) {
	db := newTestDb(t)

	first := newTestClient()
	if err := db.NewClient(first); err != nil {
		t.Fatal(err)
	}
	second := newTestClient()
	if err := db.NewClient(second); err != nil {
		t.Fatal(err)
	}
	third := newTestClient()
	if err := db.NewClient(third); err != nil {
		t.Fatal(err)
	}
	if first.Id != 1 || second.Id != 2 || third.Id != 3 {
		t.Fatalf("unexpected ids: %d %d %d", first.Id, second.Id, third.Id)
	}

	// Ids behave like a database auto-increment column: deleting a client must
	// not make its id available again.
	if err := db.DelClient(second.Id); err != nil {
		t.Fatal(err)
	}
	fourth := newTestClient()
	if err := db.NewClient(fourth); err != nil {
		t.Fatal(err)
	}
	if fourth.Id != 4 {
		t.Fatalf("expected id 4 after deletion, got %d", fourth.Id)
	}
}

func TestNoStoreClientDoesNotConsumePersistedId(t *testing.T) {
	db := newTestDb(t)

	first := newTestClient()
	if err := db.NewClient(first); err != nil {
		t.Fatal(err)
	}

	temporary := newTestClient()
	temporary.NoStore = true
	if err := db.NewClient(temporary); err != nil {
		t.Fatal(err)
	}
	if temporary.Id >= 0 {
		t.Fatalf("expected a negative id for a NoStore client, got %d", temporary.Id)
	}

	// The temporary client must not advance the persisted auto-increment id.
	second := newTestClient()
	if err := db.NewClient(second); err != nil {
		t.Fatal(err)
	}
	if second.Id != 2 {
		t.Fatalf("expected id 2, got %d", second.Id)
	}
}

func TestClientIdRestartsFromOneWhenAllDeleted(t *testing.T) {
	db := newTestDb(t)

	first := newTestClient()
	second := newTestClient()
	for _, c := range []*Client{first, second} {
		if err := db.NewClient(c); err != nil {
			t.Fatal(err)
		}
	}

	// A NoStore client must not keep the persisted counter alive.
	temporary := newTestClient()
	temporary.NoStore = true
	if err := db.NewClient(temporary); err != nil {
		t.Fatal(err)
	}

	if err := db.DelClient(first.Id); err != nil {
		t.Fatal(err)
	}
	if err := db.DelClient(second.Id); err != nil {
		t.Fatal(err)
	}

	// The data file is empty now, so numbering starts from 1 again.
	third := newTestClient()
	if err := db.NewClient(third); err != nil {
		t.Fatal(err)
	}
	if third.Id != 1 {
		t.Fatalf("expected id 1 after deleting all clients, got %d", third.Id)
	}
}

func TestLoadEmptyClientFileStartsAtOne(t *testing.T) {
	db := newTestDb(t)
	if err := os.WriteFile(filepath.Join(db.JsonDb.RunPath, "conf", "clients.json"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	db.JsonDb.LoadClientFromJsonFile()

	first := newTestClient()
	if err := db.NewClient(first); err != nil {
		t.Fatal(err)
	}
	if first.Id != 1 {
		t.Fatalf("expected id 1 for an empty data file, got %d", first.Id)
	}
}

func TestTaskAndHostIdsRestartFromOneWhenAllDeleted(t *testing.T) {
	db := newTestDb(t)

	task1 := &Tunnel{Mode: "tcp"}
	task2 := &Tunnel{Mode: "tcp"}
	for _, task := range []*Tunnel{task1, task2} {
		if err := db.NewTask(task); err != nil {
			t.Fatal(err)
		}
	}
	for _, task := range []*Tunnel{task1, task2} {
		if err := db.DelTask(task.Id); err != nil {
			t.Fatal(err)
		}
	}
	task3 := &Tunnel{Mode: "tcp"}
	if err := db.NewTask(task3); err != nil {
		t.Fatal(err)
	}
	if task3.Id != 1 {
		t.Fatalf("expected task id 1 after deleting all tasks, got %d", task3.Id)
	}

	host1 := &Host{Host: "a.example.com", Scheme: "http"}
	host2 := &Host{Host: "b.example.com", Scheme: "http"}
	for _, host := range []*Host{host1, host2} {
		if err := db.NewHost(host); err != nil {
			t.Fatal(err)
		}
	}
	for _, host := range []*Host{host1, host2} {
		if err := db.DelHost(host.Id); err != nil {
			t.Fatal(err)
		}
	}
	host3 := &Host{Host: "c.example.com", Scheme: "http"}
	if err := db.NewHost(host3); err != nil {
		t.Fatal(err)
	}
	if host3.Id != 1 {
		t.Fatalf("expected host id 1 after deleting all hosts, got %d", host3.Id)
	}
}

func TestNewTaskIdsOnlyMoveForward(t *testing.T) {
	db := newTestDb(t)

	first := &Tunnel{Mode: "tcp"}
	if err := db.NewTask(first); err != nil {
		t.Fatal(err)
	}
	second := &Tunnel{Mode: "tcp"}
	if err := db.NewTask(second); err != nil {
		t.Fatal(err)
	}
	if first.Id != 1 || second.Id != 2 {
		t.Fatalf("unexpected task ids: %d %d", first.Id, second.Id)
	}
	if err := db.DelTask(first.Id); err != nil {
		t.Fatal(err)
	}
	third := &Tunnel{Mode: "tcp"}
	if err := db.NewTask(third); err != nil {
		t.Fatal(err)
	}
	if third.Id != 3 {
		t.Fatalf("expected task id 3 after deletion, got %d", third.Id)
	}

	temporary := &Tunnel{Mode: "tcp", NoStore: true}
	if err := db.NewTask(temporary); err != nil {
		t.Fatal(err)
	}
	if temporary.Id >= 0 {
		t.Fatalf("expected a negative id for a NoStore task, got %d", temporary.Id)
	}
}

func TestNewHostIdsOnlyMoveForward(t *testing.T) {
	db := newTestDb(t)

	first := &Host{Host: "a.example.com", Scheme: "http"}
	if err := db.NewHost(first); err != nil {
		t.Fatal(err)
	}
	second := &Host{Host: "b.example.com", Scheme: "http"}
	if err := db.NewHost(second); err != nil {
		t.Fatal(err)
	}
	if first.Id != 1 || second.Id != 2 {
		t.Fatalf("unexpected host ids: %d %d", first.Id, second.Id)
	}
	if err := db.DelHost(first.Id); err != nil {
		t.Fatal(err)
	}
	third := &Host{Host: "c.example.com", Scheme: "http"}
	if err := db.NewHost(third); err != nil {
		t.Fatal(err)
	}
	if third.Id != 3 {
		t.Fatalf("expected host id 3 after deletion, got %d", third.Id)
	}

	temporary := &Host{Host: "tmp.example.com", Scheme: "http", NoStore: true}
	if err := db.NewHost(temporary); err != nil {
		t.Fatal(err)
	}
	if temporary.Id >= 0 {
		t.Fatalf("expected a negative id for a NoStore host, got %d", temporary.Id)
	}
}

func TestSortClientsDefaultsToIdDesc(t *testing.T) {
	list := []*Client{{Id: 2}, {Id: 7}, {Id: 1}}
	SortClients(list, "", "")
	got := []int{list[0].Id, list[1].Id, list[2].Id}
	want := []int{7, 2, 1}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestNewClientConcurrentIdsAreUnique(t *testing.T) {
	db := newTestDb(t)

	const count = 50
	ids := make(chan int, count)
	errs := make(chan error, count)
	done := make(chan struct{}, count)
	for i := 0; i < count; i++ {
		go func() {
			c := newTestClient()
			if err := db.NewClient(c); err != nil {
				errs <- err
			} else {
				ids <- c.Id
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < count; i++ {
		<-done
	}
	close(ids)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	seen := make(map[int]bool, count)
	for id := range ids {
		if id < 1 || id > count {
			t.Fatalf("id %d out of expected range", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = true
	}
	if len(seen) != count {
		t.Fatalf("expected %d unique ids, got %d", count, len(seen))
	}
}
