package codebase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lucasew/workspaced/internal/afterwait"
	"github.com/lucasew/workspaced/internal/atomicfile"
	"github.com/lucasew/workspaced/internal/checks/lint"
	"github.com/lucasew/workspaced/internal/checks/review"
	"github.com/lucasew/workspaced/internal/cmdarg"
	"github.com/lucasew/workspaced/internal/taskui"
	"github.com/lucasew/workspaced/pkg/logging"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/owenrumney/go-sarif/v2/sarif"
)

type Lint struct {
	Format cmd.EnumArg[cmdarg.LintFormat] `short:"f" long:"format" help:"Output format (table, sarif)" default:"table"`
	Review cmd.Flag                       `long:"review" help:"Post GitHub Actions annotations for findings on the relevant diff"`
	path   cmd.WorkDirArg
}

func (Lint) Description() string {
	return "Run linters on the specified path (defaults to current directory)"
}

func (c *Lint) Run(ctx context.Context) error {
	path, err := filepath.Abs(c.path.Value())
	if err != nil {
		return err
	}

	var report *sarif.Report
	return taskui.Run(ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, "codebase:lint", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
			s.Update("running linters")
			var err error
			report, err = lint.RunAll(ctx, path)
			return err
		})
		format := c.Format.Value()
		doReview := c.Review.Value()
		afterwait.Register(ctx, func() error {
			if report == nil {
				return nil
			}
			saveSarifToCI(ctx, report)
			if doReview {
				if err := review.AnnotateIfApplicable(ctx, report, review.AnnotateOptions{Root: path}); err != nil {
					return err
				}
			}
			return printReport(report, format)
		})
		return nil
	})
}

func saveSarifToCI(ctx context.Context, report *sarif.Report) {
	logger := logging.GetLogger(ctx)
	sarifEnvVars := []string{"MISE_CI_SARIF_OUTPUT_DIR"}
	for _, envVar := range sarifEnvVars {
		if outputDir := os.Getenv(envVar); outputDir != "" {
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				logger.Warn("failed to create SARIF output directory", "output_dir", outputDir, "error", err)
				continue
			}

			sarifPath := filepath.Join(outputDir, "lint.sarif")
			if err := writeSarifAtomic(sarifPath, report); err != nil {
				logger.Warn("failed to write SARIF report", "sarif_path", sarifPath, "error", err)
			}
		}
	}
}

func printReport(report *sarif.Report, format cmdarg.LintFormat) error {
	switch format {
	case cmdarg.LintSARIF:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	default:
		return printTable(report)
	}
}

func printTable(report *sarif.Report) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "TOOL\tLEVEL\tFILE:LINE\tMESSAGE"); err != nil {
		return err
	}

	for _, run := range report.Runs {
		toolName := run.Tool.Driver.Name

		if run.Tool.Driver.Name != "" {
			toolName = run.Tool.Driver.Name
		} else if run.Tool.Driver.InformationURI != nil {
			toolName = *run.Tool.Driver.InformationURI
		}

		for _, res := range run.Results {
			file := ""
			line := 0
			msg := ""

			if res.Message.Text != nil {
				msg = *res.Message.Text
			}

			if len(res.Locations) > 0 {
				loc := res.Locations[0].PhysicalLocation
				if loc != nil {
					if loc.ArtifactLocation != nil && loc.ArtifactLocation.URI != nil {
						file = *loc.ArtifactLocation.URI
					}
					if loc.Region != nil && loc.Region.StartLine != nil {
						line = *loc.Region.StartLine
					}
				}
			}

			fileLine := file
			if line > 0 {
				fileLine = fmt.Sprintf("%s:%d", file, line)
			}

			level := "unknown"
			if res.Level != nil {
				level = *res.Level
			}

			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", toolName, level, fileLine, msg); err != nil {
				return err
			}
		}
	}
	return w.Flush()
}

func writeSarifAtomic(path string, report *sarif.Report) (err error) {
	f, err := atomicfile.Create(path, 0)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Abort()) }()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(report); err != nil {
		return err
	}
	return f.Commit()
}
