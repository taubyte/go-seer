package seer

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

func (s *Seer) Batch(queries ...*Query) *Batch {
	b := &Batch{
		queries: make([]*Query, len(queries)),
	}

	copy(b.queries, queries)

	return b
}

func (s *Seer) Sync() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	for docName, doc := range s.documents {
		f, err := s.fs.OpenFile(docName, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0640)
		if err != nil {
			return fmt.Errorf("Opening %s failed with %s", docName, err.Error())
		}
		defer f.Close()

		enc := yaml.NewEncoder(f)
		err = enc.Encode(doc)
		if err != nil {
			return fmt.Errorf("Encoding data to %s failed with %s", docName, err.Error())
		}
	}
	return nil
}

func (s *Seer) Get(name string) *Query {
	return s.nilQuery().Get(name)
}

func (s *Seer) List() ([]string, error) {

	list, err := afero.ReadDir(s.fs, "/")
	if err != nil {
		return nil, fmt.Errorf("Listing seer's root failed with %s", err.Error())
	}

	out := make([]string, 0)
	for _, s := range list {
		name := s.Name()
		if s.IsDir() {
			out = append(out, name)
		} else if strings.HasSuffix(name, ".yaml") == true {
			out = append(out, strings.TrimSuffix(name, ".yaml"))
		}
	}

	return out, nil
}

func (s *Seer) nilQuery() *Query {
	return &Query{
		seer:   s,
		ops:    make([]op, 0),
		errors: make([]error, 0),
	}
}

func (s *Seer) loadYamlDocument(path string) (*yaml.Node, error) {
	f, err := s.fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Opening yaml file %s failed with %s", path, err.Error())
	}

	root_node := &yaml.Node{}
	yaml_decoder := yaml.NewDecoder(f)
	err = yaml_decoder.Decode(root_node)
	if err != nil {
		return nil, fmt.Errorf("Processing yaml file %s failed with %s", path, err.Error())
	}

	s.documents[path] = root_node
	return root_node, nil
}
