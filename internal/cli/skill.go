package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/raphaelCamblong/duty/internal/app"
	"github.com/raphaelCamblong/duty/internal/fetch"
)

const (
	skillURL          = "https://duty-cli.xyz/skill.md"
	skillExample      = "  duty skill\n  duty skill install claude"
	skillInstallUsage = "usage: duty skill install [claude|codex|gemini] [--user] [--force] [--offline]"
)

// skillCtx is the shared context the skill command constructors build from.
type skillCtx struct {
	app     app.App
	fetcher fetch.Fetcher
	cwd     string
	home    string
	out     io.Writer
}

// newSkillCmd fetches the skill from duty-cli.xyz and falls back silently to
// the embedded copy on failure; --offline skips the network entirely.
func newSkillCmd(sc skillCtx) *cobra.Command {
	var offline bool
	cmd := &cobra.Command{
		Use:     "skill",
		Short:   "print the duty agent skill, or install it into a harness",
		Example: skillExample,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) != 0 {
				return unknownCommand(c, args[0])
			}
			_, err := sc.out.Write(sc.app.Skill(sc.fetcher, skillURL, offline))
			return err
		},
	}
	cmd.PersistentFlags().BoolVar(&offline, "offline", false, "skip the network fetch; use the embedded copy")
	cmd.AddCommand(newSkillInstallCmd(sc, &offline))
	return cmd
}

func newSkillInstallCmd(sc skillCtx, offline *bool) *cobra.Command {
	var (
		user  bool
		force bool
	)
	cmd := &cobra.Command{
		Use:     "install [claude|codex|gemini]",
		Short:   "install the duty skill into an agent harness",
		Example: "  duty skill install claude",
		RunE: func(c *cobra.Command, args []string) error {
			target, err := resolveTarget(c, args)
			if err != nil {
				return err
			}
			spec := app.Install{Target: target, Cwd: sc.cwd, Home: sc.home, User: user, Force: force}
			path, err := sc.app.InstallSkill(spec, sc.app.Skill(sc.fetcher, skillURL, *offline))
			if err != nil {
				return err
			}
			fmt.Fprintf(sc.out, "installed %s skill → %s\n", target, path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&user, "user", false, "install for claude in your home directory, not this repo")
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing install")
	return cmd
}

func resolveTarget(cmd *cobra.Command, args []string) (app.Target, error) {
	if len(args) > 1 {
		return "", errors.New(skillInstallUsage)
	}
	if len(args) == 1 {
		return app.ParseTarget(args[0])
	}
	if !interactive(cmd) {
		return "", errors.New("name a target: claude, codex or gemini")
	}
	return selectTarget()
}

func selectTarget() (app.Target, error) {
	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Install the duty skill for which agent?").
				Options(
					huh.NewOption("Claude Code (.claude/skills)", string(app.Claude)),
					huh.NewOption("Codex (AGENTS.md)", string(app.Codex)),
					huh.NewOption("Gemini (GEMINI.md)", string(app.Gemini)),
				).
				Value(&choice),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return app.ParseTarget(choice)
}

func interactive(cmd *cobra.Command) bool {
	return isTTY(cmd.InOrStdin()) && isTTY(cmd.OutOrStdout())
}

func isTTY(v any) bool {
	f, ok := v.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
