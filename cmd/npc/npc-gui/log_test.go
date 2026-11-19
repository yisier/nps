package main

import "testing"

func TestFilteredLogs(t *testing.T) {
	a := &App{}
	a.ui.clients = []ShortClient{
		{Name: "one", Addr: "1.1.1.1:8000", Key: "k1"},
		{Name: "two", Addr: "2.2.2.2:8000", Key: "k2"},
	}
	a.ui.logs = []ConnectionLog{
		{Message: "[I] [k1] start vkey:k1"},
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

	// 下拉框显示【备注-vkey】，没有备注时只显示 vkey。
	names, ids := a.logOptions()
	if len(names) != 3 || names[1] != "one-k1" || names[2] != "two-k2" || ids[1] != "1.1.1.1:8000|k1" {
		t.Fatalf("log options = %v, %v", names, ids)
	}

	// 同一地址有多个客户端在运行时，不能用地址过滤，避免串日志。
	a.ui.clients = []ShortClient{
		{Name: "a", Addr: "1.1.1.1:8000", Key: "k1", Status: "connected"},
		{Name: "b", Addr: "1.1.1.1:8000", Key: "k2", Status: "connected"},
	}
	a.ui.logs = []ConnectionLog{
		{Message: "[I] [k1] start vkey:k1"},
		{Message: "[I] [client.go:104] Successful connection with server 1.1.1.1:8000"},
		{Message: "[E] [control.go:269] EOF"},
	}
	a.ui.logFilterID = "1.1.1.1:8000|k1"
	if got := a.filteredLogs(); len(got) != 1 {
		t.Fatalf("shared address logs = %d, want only the prefixed line: %+v", len(got), got)
	}

	// 同地址只剩一个在运行时，可以安全地用地址匹配全局日志。
	a.ui.clients[1].Status = "stopped"
	if got := a.filteredLogs(); len(got) != 2 {
		t.Fatalf("sole running client logs = %d, want 2: %+v", len(got), got)
	}
}
