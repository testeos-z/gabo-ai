package main

import (
	"context"
	"os"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/osfs"
	"github.com/testeos-z/gbai-team/apps/cli/internal/skills"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/initcmd"
)

// initFactory builds the init use case lazily, so help and usage errors never
// touch the filesystem. The project is the git top-level of the working
// directory, or the directory itself.
func initFactory(runner kernel.Runner, version string) initcmd.Factory {
	return func(ctx context.Context) (*initcmd.UseCase, error) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		// skills.Files excludes gabo-reviewer, which never enters the repository (D20).
		embedded, err := skills.Files()
		if err != nil {
			return nil, err
		}
		files := make([]initcmd.File, 0, len(embedded))
		for _, f := range embedded {
			files = append(files, initcmd.File{Path: f.Path, Data: f.Data})
		}
		return initcmd.NewUseCase(osfs.FS{}, files, initcmd.Settings{
			ProjectDir: config.ProjectRoot(ctx, runner, wd),
			Version:    version,
		}), nil
	}
}
