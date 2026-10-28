package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel/osfs"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/setupprompt"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/check"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/initcmd"
)

// reviewerSkillDir is the user-level location of the gabo-reviewer skill
// (TRD 9.4): ${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer. A
// relative XDG_DATA_HOME is ignored, as the XDG spec requires.
func reviewerSkillDir(getenv func(string) string) (string, error) {
	data := getenv("XDG_DATA_HOME")
	if !filepath.IsAbs(data) {
		home := getenv("HOME")
		if !filepath.IsAbs(home) {
			return "", errors.New("reviewer skill dir: set XDG_DATA_HOME or HOME to an absolute path")
		}
		data = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(data, "gabo", "skills", "gabo-reviewer"), nil
}

// initFactory builds the init run lazily, so help and usage errors never
// touch the filesystem. The project is the git top-level of the working
// directory, or the directory itself.
func initFactory(runner kernel.Runner, version string) initcmd.Factory {
	return func(ctx context.Context) (*initcmd.Init, error) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		dir, err := reviewerSkillDir(os.Getenv)
		if err != nil {
			return nil, err
		}
		// skills.Files excludes gabo-reviewer, which never enters the repository (D20).
		embedded, err := skills.Files()
		if err != nil {
			return nil, err
		}
		reviewer, err := skills.ReviewerFiles()
		if err != nil {
			return nil, err
		}
		project := config.ProjectRoot(ctx, runner, wd)
		return &initcmd.Init{
			Reviewer: initcmd.NewReviewer(osfs.FS{}, toFiles(reviewer, "gabo-reviewer/"), initcmd.ReviewerSettings{
				Dir: dir, ProjectDir: project, Version: version,
			}),
			Catalog: initcmd.NewUseCase(osfs.FS{}, toFiles(embedded, ""), initcmd.Settings{
				ProjectDir: project, Version: version, SetupPrompt: setupprompt.Text,
			}),
		}, nil
	}
}

// checkFactory builds the check run lazily. The project is the git top-level
// of the working directory, or the directory itself; GABO_UPGRADE_PR=1 (set
// by the reviewer scripts and CI on an upgrade PR) validates only the config.
func checkFactory(runner kernel.Runner, version string, getenv func(string) string, getwd func() (string, error)) check.Factory {
	return func(ctx context.Context) (*check.UseCase, error) {
		wd, err := getwd()
		if err != nil {
			return nil, err
		}
		project := config.ProjectRoot(ctx, runner, wd)
		return check.NewUseCase(os.DirFS(project), osfs.FS{}, check.Settings{
			Version:   version,
			Layers:    config.LayerPaths(getenv, project),
			UpgradePR: getenv("GABO_UPGRADE_PR") == "1",
			Former:    config.FormerNames(getenv, project),
		}), nil
	}
}

// toFiles converts embedded files, trimming prefix from each path.
func toFiles(embedded []skills.File, prefix string) []initcmd.File {
	files := make([]initcmd.File, 0, len(embedded))
	for _, f := range embedded {
		files = append(files, initcmd.File{Path: strings.TrimPrefix(f.Path, prefix), Data: f.Data})
	}
	return files
}
