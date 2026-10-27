package main

import (
	"context"
	"os"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/osfs"
	"github.com/testeos-z/gbai-team/apps/cli/internal/skills"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/initcmd"
)

// newInitUseCase builds the init use case lazily, so help and flag errors
// never touch the filesystem.
func newInitUseCase(context.Context) (*initcmd.UseCase, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	embedded, err := skills.Files()
	if err != nil {
		return nil, err
	}
	files := make([]initcmd.File, 0, len(embedded))
	for _, f := range embedded {
		files = append(files, initcmd.File{Path: f.Path, Data: f.Data})
	}
	return initcmd.NewUseCase(osfs.FS{}, files, initcmd.Settings{Home: home, ProjectDir: wd}), nil
}
