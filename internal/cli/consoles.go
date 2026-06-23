package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thathaneydude/unifi/unifi"
)

// console is the minimal identity of a UniFi console (a "host" in the Site
// Manager API) used both to render `consoles list` and to resolve a --console
// value to the id the cloud connector expects.
//
// ID is the 71-character host id returned by GET /v1/hosts — confirmed to be the
// value the connector path /v1/connector/consoles/<id>/... requires (NOT the
// 36-character hardwareId, which the connector rejects with 403).
type console struct {
	ID        string
	Name      string // reportedState.name (falls back to hostname)
	Model     string // reportedState.hardware.name, e.g. "UniFi Dream Machine SE"
	Shortname string // reportedState.hardware.shortname, e.g. "UDMPROSE"
	IP        string // ipAddress
	Owner     bool
}

// consoleView is the JSON projection emitted by `consoles list`. It is a compact,
// LLM-friendly shape rather than the large raw /v1/hosts record (which embeds a
// huge reportedState). Field selection/redaction/limit apply to this view.
type consoleView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Model     string `json:"model"`
	Shortname string `json:"shortname,omitempty"`
	IP        string `json:"ip,omitempty"`
	Owner     bool   `json:"owner"`
}

// parseHostsPage reads one page of the {data:[...], nextToken} envelope returned
// by the Site Manager API's GET /v1/hosts, returning the consoles on the page and
// the pagination token ("" when there are no more pages).
func parseHostsPage(body []byte) ([]console, string, error) {
	var env struct {
		Data []struct {
			ID            string `json:"id"`
			IPAddress     string `json:"ipAddress"`
			Owner         bool   `json:"owner"`
			ReportedState struct {
				Name     string `json:"name"`
				Hostname string `json:"hostname"`
				Hardware struct {
					Name      string `json:"name"`
					Shortname string `json:"shortname"`
				} `json:"hardware"`
			} `json:"reportedState"`
		} `json:"data"`
		NextToken string `json:"nextToken"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, "", err
	}
	out := make([]console, 0, len(env.Data))
	for _, d := range env.Data {
		name := d.ReportedState.Name
		if name == "" {
			name = d.ReportedState.Hostname
		}
		out = append(out, console{
			ID:        d.ID,
			Name:      name,
			Model:     d.ReportedState.Hardware.Name,
			Shortname: d.ReportedState.Hardware.Shortname,
			IP:        d.IPAddress,
			Owner:     d.Owner,
		})
	}
	return out, env.NextToken, nil
}

// listConsoles fetches every console on the account, following nextToken
// pagination until exhausted. conn must be an account-level connection
// (unifi.Account) so the path resolves to https://api.ui.com/v1/hosts.
func listConsoles(ctx context.Context, conn *unifi.Conn) ([]console, error) {
	var all []console
	token := ""
	for {
		q := map[string]string{"pageSize": "100"}
		if token != "" {
			q["nextToken"] = token
		}
		// A synthetic operation reuses Execute's auth/transport/error handling.
		// App is irrelevant: the account prefix ignores it.
		op := Operation{App: unifi.AppNetwork, ID: "listHosts", Method: http.MethodGet, Path: "/v1/hosts"}
		body, status, err := Execute(ctx, conn, op, Values{Path: map[string]string{}, Query: q})
		if err != nil {
			return nil, err
		}
		if status < 200 || status >= 300 {
			return nil, NewAPIError("listHosts", status, body)
		}
		page, next, perr := parseHostsPage(body)
		if perr != nil {
			return nil, NewUsageError("parsing /v1/hosts response: " + perr.Error())
		}
		all = append(all, page...)
		if next == "" {
			break
		}
		token = next
	}
	return all, nil
}

func consoleLabels(cs []console) []string {
	labels := make([]string, 0, len(cs))
	for _, c := range cs {
		label := c.Name
		if c.Shortname != "" {
			label += " (" + c.Shortname + ")"
		}
		labels = append(labels, label)
	}
	return labels
}

// selectConsole resolves a console id from a list and a wanted value, mirroring
// selectSite. An empty want auto-selects when exactly one console exists;
// otherwise want is matched case-insensitively against a console's id, name,
// shortname, or model.
func selectConsole(cs []console, want string) (string, error) {
	if want == "" {
		switch len(cs) {
		case 0:
			return "", NewUsageError("no consoles found on this account")
		case 1:
			return cs[0].ID, nil
		default:
			return "", NewUsageError("multiple consoles; pass --console <name|model|id> — have: " + strings.Join(consoleLabels(cs), ", "))
		}
	}
	w := strings.ToLower(want)
	var matches []string
	for _, c := range cs {
		if strings.ToLower(c.ID) == w || strings.ToLower(c.Name) == w ||
			strings.ToLower(c.Shortname) == w || strings.ToLower(c.Model) == w {
			matches = append(matches, c.ID)
		}
	}
	switch len(matches) {
	case 0:
		return "", NewUsageError("no console matches --console " + strconv.Quote(want) + " — have: " + strings.Join(consoleLabels(cs), ", "))
	case 1:
		return matches[0], nil
	default:
		return "", NewUsageError("ambiguous --console " + strconv.Quote(want) + " matches multiple consoles")
	}
}

// newConsolesCommand returns `unifi consoles`, whose `list` subcommand enumerates
// every console on the account via the cloud Site Manager API (GET /v1/hosts).
// It needs only the shared API key — not --host/--console-id — because it is
// discovering consoles rather than talking to one.
func newConsolesCommand(
	stdout io.Writer,
	accountConn func() (*unifi.Conn, error),
	format func() Format,
	render func() RenderOptions,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consoles",
		Short: "List UniFi consoles on your account (cloud Site Manager API)",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return NewUsageError("no subcommand; run 'unifi consoles list'")
			}
			return NewUsageError(fmt.Sprintf("unknown subcommand %q; run 'unifi consoles list'", args[0]))
		},
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List all consoles on the account (id, name, model, ip)",
		Long: "Enumerate every UniFi console on the account via the cloud Site " +
			"Manager API. The 'id' field is the value to pass as --console-id (or " +
			"--console <name>) to target a console. Requires the shared API key " +
			"(--api-key / UNIFI_API_KEY); --host/--console-id are not used.",
		RunE: func(c *cobra.Command, _ []string) error {
			conn, err := accountConn()
			if err != nil {
				return err
			}
			consoles, err := listConsoles(c.Context(), conn)
			if err != nil {
				return err
			}
			// console and consoleView share identical fields; the view only adds
			// the JSON tags for the emitted shape, so a direct conversion suffices.
			views := make([]consoleView, 0, len(consoles))
			for _, cc := range consoles {
				views = append(views, consoleView(cc))
			}
			body, err := json.Marshal(views)
			if err != nil {
				return err
			}
			f := format()
			// Mirror runOperation: raw passthrough skips transforms.
			if f != FormatRaw {
				body = ApplyTransforms(body, render())
			}
			return WriteResult(stdout, f, body)
		},
	}
	cmd.AddCommand(list)
	return cmd
}
