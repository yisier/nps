package main

import "testing"

func TestFilteredLogs(t *testing.T) {
	a := &App{}
	a.ui.clients = []ShortClient{
		{Name: "one", Addr: "1.1.1.1:8000", Key: "k1"},
		{Name: "two", Addr: "2.2.2.2:8000", Key: "k2"},
	}
	a.ui.logs = []ConnectionLog{
		{Message: "[I] [1.1.1.1:8000|k1] start vkey:k1"},
		{Message: "[I] [client.go:104] Successful connection with server 1.1.1.1:8000"},
		{Message: "[E] [control.go:269] EOF"},
		{Message: "[I] [client.go:104] Successful connection with server 2.2.2.2:8000"},
	}

	a.ui.logFilterID = "1.1.1.1:8000|k1"
	got := a.filteredLogs()
	if len(got) != 2 {
		t.Fatalf("filtered logs = %d, want 2: %+v", len(got), got)
	}
	for _, l := range got {
		if l.Message == "[E] [control.go:269] EOF" {
			t.Fatal("unattributed log shown for a client")
		}
	}

	a.ui.logFilterID = ""
	if got := a.filteredLogs(); len(got) != 4 {
		t.Fatalf("all logs = %d, want 4", len(got))
	}
}
