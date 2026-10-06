package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/usecase"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type updateFlags struct {
	dryRun bool
	list   bool
	only   string
}

func runUpdateCommand() *cobra.Command {
	var flags updateFlags
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the global toolchain envctl provisions (Node/npm, Python and Go tools)",
		Long: "Updates the global tools the manifests install through a user-local mechanism:\n" +
			"mise runtimes, npm globals, uv-managed Python tools and go-installed tools.\n\n" +
			"Applies without prompting: every one of those is user-local, needs no sudo and is\n" +
			"reversible, and the report records the previous version. Use --dry-run to preview.\n\n" +
			"OS package managers are deliberately NOT automated. Upgrading a subset through\n" +
			"pacman is a partial upgrade, which Arch forbids; apt and winget follow the system\n" +
			"update instead. Tools installed by the bootstrap rather than the manifests\n" +
			"(golangci-lint, go, gh, yq, delta, fd, fzf) are out of scope for the same reason:\n" +
			"golangci-lint installs through curl|sh and the rest belong to the OS.",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runUpdateProvisioning(flags)
		},
	}
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Print the command each update would run and change nothing")
	cmd.Flags().BoolVar(&flags.list, "list", false, "List the automatable inventory without querying any registry")
	cmd.Flags().StringVar(&flags.only, "only", "", "Narrow to one group: mise, npm, uv or go")
	return cmd
}

func runUpdateProvisioning(flags updateFlags) {
	ctx := context.Background()

	packages, err := appCtx.ManifestRepo.LoadPackages()
	if err != nil {
		pterm.Error.Printf("  • failed to load packages manifest: %v\n", err)
		return
	}
	lsps, err := appCtx.ManifestRepo.LoadLSPs()
	if err != nil {
		pterm.Error.Printf("  • failed to load LSP manifest: %v\n", err)
		return
	}

	opts := usecase.UpdateOptions{DryRun: flags.dryRun, List: flags.list}
	if flags.only != "" {
		group := usecase.UpdateGroup(flags.only)
		switch group {
		case usecase.GroupMise, usecase.GroupNpm, usecase.GroupUV, usecase.GroupGo:
			opts.Only = group
		default:
			pterm.Error.Printf("  • unknown group %q: use mise, npm, uv or go\n", flags.only)
			return
		}
	}

	result, err := appCtx.UpdateUC.Execute(ctx, usecase.UpdateInventory{
		Packages: applicablePackages(packages),
		LSPs:     applicableLSPs(lsps),
	}, opts)
	if err != nil {
		pterm.Error.Printf("  • update failed: %v\n", err)
		return
	}

	printUpdateInventory(result)

	if flags.list {
		return
	}
	if flags.dryRun {
		pterm.Info.Println("  • dry run: nothing was changed")
		return
	}
	printUpdateOutcomes(result)
}

// applicablePackages keeps only the entries the running platform would install,
// so a Windows-only tool never shows up in a Linux inventory.
func applicablePackages(packages []entity.Package) []entity.Package {
	var out []entity.Package
	for _, p := range packages {
		if entity.MatchesOS(p.OS) {
			out = append(out, p)
		}
	}
	return out
}

func applicableLSPs(lsps []entity.LSP) []entity.LSP {
	var out []entity.LSP
	for _, l := range lsps {
		if entity.MatchesOS(l.OS) {
			out = append(out, l)
		}
	}
	return out
}

func printUpdateInventory(result *usecase.UpdateResult) {
	if len(result.Planned) == 0 {
		pterm.Info.Println("  • nothing in the manifests is updated by envctl on this platform")
		return
	}
	var byGroup = map[usecase.UpdateGroup][]usecase.UpdateCandidate{}
	for _, c := range result.Planned {
		byGroup[c.Group] = append(byGroup[c.Group], c)
	}
	for _, group := range []usecase.UpdateGroup{usecase.GroupMise, usecase.GroupNpm, usecase.GroupUV, usecase.GroupGo} {
		items := byGroup[group]
		if len(items) == 0 {
			continue
		}
		PrintSection(fmt.Sprintf("%s (%d tools)", group, len(items)))
		for _, c := range items {
			pterm.Success.Println(fmt.Sprintf("    • %-34s %s", c.DisplayName(), c.Command))
		}
	}
}

func printUpdateOutcomes(result *usecase.UpdateResult) {
	for _, o := range result.Applied {
		pterm.Success.Println(fmt.Sprintf("  • %-30s %s -> %s", o.ID, o.From, o.To))
	}
	for _, o := range result.Current {
		pterm.Info.Println(fmt.Sprintf("  • %-30s current (%s)", o.ID, o.From))
	}
	for _, o := range result.Skipped {
		pterm.Warning.Println(fmt.Sprintf("  • %-30s skipped: %s", o.ID, o.Detail))
	}
	for _, o := range result.Failed {
		pterm.Error.Println(fmt.Sprintf("  • %-30s FAILED: %s", o.ID, strings.TrimSpace(o.Detail)))
	}
	pterm.Info.Printf("  • updated %d, current %d, skipped %d, failed %d (OS packages are never automated)\n",
		len(result.Applied), len(result.Current), len(result.Skipped), len(result.Failed))
}
