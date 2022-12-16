package seer

import (
	"fmt"

	"github.com/spf13/afero"
)

type Option func(s *Seer) error

func SystemFS(path string) Option {
	return func(s *Seer) error {
		if s.fs != nil {
			return fmt.Errorf("Can't combile *Fs() Options")
		}
		fs := afero.NewBasePathFs(afero.OsFs{}, path)
		_, err := fs.Stat("/")
		if err != nil {
			return fmt.Errorf("Opening repository failed with %s", err.Error())
		}
		s.fs = fs
		return nil
	}
}

func VirtualFS(fs afero.Fs, path string) Option {
	return func(s *Seer) error {
		if s.fs != nil {
			return fmt.Errorf("Can't combine *Fs() Options")
		}
		fs = afero.NewBasePathFs(fs, path)
		_, err := fs.Stat("/")
		if err != nil {
			return fmt.Errorf("Opening repository failed with %s", err.Error())
		}
		s.fs = fs
		return nil
	}
}
