package file

import (
	"sort"
	"strings"
)

func lessBool(a, b bool, asc bool) bool {
	if a == b {
		return false
	}
	if asc {
		return !a && b // false first
	}
	return a && !b // true first
}

func lessInt(a, b int, asc bool) bool {
	if asc {
		return a < b
	}
	return a > b
}

func lessInt64(a, b int64, asc bool) bool {
	if asc {
		return a < b
	}
	return a > b
}

func lessString(a, b string, asc bool) bool {
	if asc {
		return strings.ToLower(a) < strings.ToLower(b)
	}
	return strings.ToLower(a) > strings.ToLower(b)
}

// SortClients sorts clients in-place by the given field (bootstrap-table sort name).
func SortClients(list []*Client, sortField, order string) {
	if sortField == "" || len(list) < 2 {
		if sortField == "" && len(list) > 1 {
			sort.SliceStable(list, func(i, j int) bool { return list[i].Id < list[j].Id })
		}
		return
	}
	asc := order != "desc"
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		switch sortField {
		case "Id":
			return lessInt(a.Id, b.Id, asc)
		case "Remark":
			return lessString(a.Remark, b.Remark, asc)
		case "Version":
			return lessString(a.Version, b.Version, asc)
		case "VerifyKey":
			return lessString(a.VerifyKey, b.VerifyKey, asc)
		case "Addr":
			return lessString(a.Addr, b.Addr, asc)
		case "LocalAddr":
			return lessString(a.LocalAddr, b.LocalAddr, asc)
		case "InletFlow":
			return lessInt64(flowInlet(a), flowInlet(b), asc)
		case "ExportFlow":
			return lessInt64(flowExport(a), flowExport(b), asc)
		case "NowRate":
			return lessInt64(nowRate(a), nowRate(b), asc)
		case "Status":
			return lessBool(a.Status, b.Status, asc)
		case "IsConnect":
			return lessBool(a.IsConnect, b.IsConnect, asc)
		default:
			return lessInt(a.Id, b.Id, true)
		}
	})
}

func flowInlet(c *Client) int64 {
	if c == nil || c.Flow == nil {
		return 0
	}
	return c.Flow.InletFlow
}

func flowExport(c *Client) int64 {
	if c == nil || c.Flow == nil {
		return 0
	}
	return c.Flow.ExportFlow
}

func nowRate(c *Client) int64 {
	if c == nil || c.Rate == nil {
		return 0
	}
	return c.Rate.NowRate
}

// SortTunnels sorts tunnels in-place by the given field.
func SortTunnels(list []*Tunnel, sortField, order string) {
	if sortField == "" || len(list) < 2 {
		if sortField == "" && len(list) > 1 {
			sort.SliceStable(list, func(i, j int) bool { return list[i].Id < list[j].Id })
		}
		return
	}
	asc := order != "desc"
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		switch sortField {
		case "Id":
			return lessInt(a.Id, b.Id, asc)
		case "ClientId":
			return lessInt(clientIdOfTunnel(a), clientIdOfTunnel(b), asc)
		case "Remark":
			return lessString(a.Remark, b.Remark, asc)
		case "Client.VerifyKey", "VerifyKey":
			return lessString(clientVkeyOfTunnel(a), clientVkeyOfTunnel(b), asc)
		case "Mode":
			return lessString(a.Mode, b.Mode, asc)
		case "Port":
			return lessInt(a.Port, b.Port, asc)
		case "Target":
			return lessString(targetStrOfTunnel(a), targetStrOfTunnel(b), asc)
		case "Password":
			return lessString(a.Password, b.Password, asc)
		case "Status":
			return lessBool(a.Status, b.Status, asc)
		case "RunStatus":
			return lessBool(a.RunStatus, b.RunStatus, asc)
		case "IsConnect", "Client.IsConnect":
			return lessBool(clientConnectOfTunnel(a), clientConnectOfTunnel(b), asc)
		default:
			return lessInt(a.Id, b.Id, true)
		}
	})
}

func clientIdOfTunnel(t *Tunnel) int {
	if t == nil || t.Client == nil {
		return 0
	}
	return t.Client.Id
}

func clientVkeyOfTunnel(t *Tunnel) string {
	if t == nil || t.Client == nil {
		return ""
	}
	return t.Client.VerifyKey
}

func clientConnectOfTunnel(t *Tunnel) bool {
	if t == nil || t.Client == nil {
		return false
	}
	return t.Client.IsConnect
}

func targetStrOfTunnel(t *Tunnel) string {
	if t == nil || t.Target == nil {
		return ""
	}
	return t.Target.TargetStr
}

// SortHosts sorts hosts in-place by the given field.
func SortHosts(list []*Host, sortField, order string) {
	if sortField == "" || len(list) < 2 {
		if sortField == "" && len(list) > 1 {
			sort.SliceStable(list, func(i, j int) bool { return list[i].Id < list[j].Id })
		}
		return
	}
	asc := order != "desc"
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		switch sortField {
		case "Id":
			return lessInt(a.Id, b.Id, asc)
		case "ClientId":
			return lessInt(clientIdOfHost(a), clientIdOfHost(b), asc)
		case "Remark":
			return lessString(a.Remark, b.Remark, asc)
		case "Client.VerifyKey", "VerifyKey":
			return lessString(clientVkeyOfHost(a), clientVkeyOfHost(b), asc)
		case "Host":
			return lessString(a.Host, b.Host, asc)
		case "Scheme":
			return lessString(a.Scheme, b.Scheme, asc)
		case "Target":
			return lessString(targetStrOfHost(a), targetStrOfHost(b), asc)
		case "Location":
			return lessString(a.Location, b.Location, asc)
		case "IsClose", "Status":
			// IsClose: false=open, true=closed — sort by open status for "Status" display
			return lessBool(a.IsClose, b.IsClose, asc)
		case "IsConnect", "Client.IsConnect":
			return lessBool(clientConnectOfHost(a), clientConnectOfHost(b), asc)
		default:
			return lessInt(a.Id, b.Id, true)
		}
	})
}

func clientIdOfHost(h *Host) int {
	if h == nil || h.Client == nil {
		return 0
	}
	return h.Client.Id
}

func clientVkeyOfHost(h *Host) string {
	if h == nil || h.Client == nil {
		return ""
	}
	return h.Client.VerifyKey
}

func clientConnectOfHost(h *Host) bool {
	if h == nil || h.Client == nil {
		return false
	}
	return h.Client.IsConnect
}

func targetStrOfHost(h *Host) string {
	if h == nil || h.Target == nil {
		return ""
	}
	return h.Target.TargetStr
}
