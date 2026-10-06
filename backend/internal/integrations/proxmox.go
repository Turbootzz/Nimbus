package integrations

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/nimbus/backend/internal/models"
)

type proxmox struct{}

func init() { Register(proxmox{}) }

func (proxmox) Kind() string { return "proxmox" }

func (proxmox) Meta() Meta {
	return Meta{
		Name:        "Proxmox VE",
		Icon:        "proxmox",
		DefaultPort: 8006,
		// An API token as USER@REALM!TOKENID=SECRET; the PVEAuditor role is enough
		AuthTypes: []string{models.IntegrationAuthToken},
		KPIs: []KPI{
			{Key: "vms", Label: "VMs"},
			{Key: "containers", Label: "LXC"},
			{Key: "cpu", Label: "CPU", Unit: "%"},
			{Key: "memory", Label: "Memory", Unit: "%"},
		},
	}
}

func proxmoxAuth(conn *Conn) authFunc {
	token := strings.TrimPrefix(conn.Creds.Token, "PVEAPIToken=")
	return func(r *http.Request) { r.Header.Set("Authorization", "PVEAPIToken="+token) }
}

func (proxmox) Test(ctx context.Context, conn *Conn) error {
	var version struct {
		Data map[string]any `json:"data"`
	}
	return getJSON(ctx, conn, "/api2/json/version", proxmoxAuth(conn), &version)
}

type proxmoxResource struct {
	Type     string  `json:"type"`
	Status   string  `json:"status"`
	Template int     `json:"template"`
	CPU      float64 `json:"cpu"` // share of maxcpu in use, 0 to 1
	MaxCPU   float64 `json:"maxcpu"`
	Mem      float64 `json:"mem"`
	MaxMem   float64 `json:"maxmem"`
}

func (proxmox) Fetch(ctx context.Context, conn *Conn) (*Payload, error) {
	var resources struct {
		Data []proxmoxResource `json:"data"`
	}
	if err := getJSON(ctx, conn, "/api2/json/cluster/resources", proxmoxAuth(conn), &resources); err != nil {
		return nil, err
	}

	var vmsUp, vms, ctUp, cts int
	var cpuUsed, cpuTotal, memUsed, memTotal float64
	for _, r := range resources.Data {
		switch {
		case r.Type == "qemu" && r.Template == 0:
			vms++
			if r.Status == "running" {
				vmsUp++
			}
		case r.Type == "lxc" && r.Template == 0:
			cts++
			if r.Status == "running" {
				ctUp++
			}
		case r.Type == "node" && r.Status == "online":
			cpuUsed += r.CPU * r.MaxCPU
			cpuTotal += r.MaxCPU
			memUsed += r.Mem
			memTotal += r.MaxMem
		}
	}
	return &Payload{KPIs: map[string]any{
		"vms":        fmt.Sprintf("%d/%d", vmsUp, vms),
		"containers": fmt.Sprintf("%d/%d", ctUp, cts),
		"cpu":        percent(cpuUsed, cpuTotal),
		"memory":     percent(memUsed, memTotal),
	}}, nil
}
