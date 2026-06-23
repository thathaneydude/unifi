package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/thathaneydude/unifi/internal/report"
)

// newReportCommand returns `unifi report`, a local (no-API) transform that
// renders a findings JSON document into a self-contained, UniFi-branded HTML
// report. It is the deliverable step of the unifi-security-assessment skill.
func newReportCommand(stdout io.Writer) *cobra.Command {
	var inPath, outPath string
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Render a findings JSON document into a self-contained HTML report",
		Long: "Render a UniFi security-assessment findings JSON document into a " +
			"single, self-contained HTML report. Reads --in (or stdin with '-') " +
			"and writes --out (or stdout).",
		RunE: func(_ *cobra.Command, _ []string) error {
			if inPath == "" {
				return NewUsageError("report requires --in <findings.json> (use '-' for stdin)")
			}

			data, err := readInput(inPath)
			if err != nil {
				return NewUsageError(err.Error())
			}
			rep, err := report.Parse(data)
			if err != nil {
				return NewUsageError(err.Error())
			}

			if outPath == "" {
				if err := report.Render(stdout, rep); err != nil {
					return fmt.Errorf("rendering report: %w", err)
				}
				return nil
			}

			f, err := os.Create(outPath)
			if err != nil {
				return NewUsageError(fmt.Sprintf("cannot write --out %q: %v", outPath, err))
			}
			if rerr := report.Render(f, rep); rerr != nil {
				_ = f.Close()
				return fmt.Errorf("rendering report: %w", rerr)
			}
			// Check Close so a deferred-write failure (e.g. disk full) surfaces.
			if cerr := f.Close(); cerr != nil {
				return fmt.Errorf("closing %q: %w", outPath, cerr)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&inPath, "in", "", "findings JSON file ('-' for stdin)")
	cmd.Flags().StringVar(&outPath, "out", "", "output HTML file (default stdout)")
	return cmd
}

// readInput reads the findings JSON from a path, or from stdin when path is "-".
func readInput(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read --in %q: %w", path, err)
	}
	return data, nil
}
