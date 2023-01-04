package seer

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/afero"
	"github.com/taubyte/utils/maps"
	pathUtils "github.com/taubyte/utils/path"

	"gopkg.in/yaml.v3"
)

type fsop func(s *Seer, path string) (string, error)

func createFolder() fsop {
	return func(s *Seer, path string) (string, error) {
		st, err := s.fs.Stat(path)
		if err != nil {
			err = s.fs.Mkdir(path, os.FileMode(0640))
			if err != nil {
				return path, fmt.Errorf("Creating folder %s failed with %s", path, err.Error())
			}
		}
		if st.IsDir() == false {
			return path, fmt.Errorf("Can't convert a %s into a folder; it's a file!", path)
		}
		return path, nil
	}
}

func createDocument() fsop {
	return func(s *Seer, path string) (string, error) {
		if _, ok := s.documents[path+".yaml"]; ok == true {
			return path + ".yaml", nil
		}

		path += ".yaml"

		newfile, err := s.fs.Create(path)
		if err != nil {
			return path, fmt.Errorf("Creating document %s failed with %s", path, err.Error())
		}
		newfile.Close()

		s.documents[path] = &yaml.Node{}

		return path, nil
	}
}

func yamlQueryFromValue(value interface{}) (*yaml.Node, error) {

	n := &yaml.Node{
		Kind: yaml.DocumentNode,
	}

	err := n.Encode(value)
	if err != nil {
		return nil, err
	}

	return n, nil

}

// Helper
func Fork(n *Query) *Query {
	return n.Fork()
}

// Copy a query ... the conly way to reuse a query.
func (n *Query) Fork() *Query {
	nq := &Query{
		seer:          n.seer,
		write:         n.write,
		requestedPath: make([]string, len(n.requestedPath)),
		ops:           make([]op, len(n.ops)),
		errors:        make([]error, 0),
	}

	copy(nq.requestedPath, n.requestedPath)
	copy(nq.ops, n.ops)

	return nq
}

func (n *Query) Set(value interface{}) *Query {
	n.ops = append(n.ops,
		op{
			opType:  opTypeSet,
			value:   value,
			handler: opSetInYaml,
		},
	)
	return n
}

func (n *Query) Delete() *Query {
	n.ops = append(n.ops,
		op{
			opType:  opTypeSet,
			handler: opDelete,
		},
	)
	return n
}

func (n *Query) Get(name string) *Query {
	n.requestedPath = append(n.requestedPath, name)
	n.ops = append(n.ops,
		op{
			opType:  opTypeGetOrCreate,
			name:    name,
			handler: opGetOrCreate,
		},
	)
	return n
}

func (n *Query) Document() *Query {
	if len(n.ops) == 0 {
		// should never happen actually, as you need to call get or set before
		n.errors = append(n.errors, errors.New("Can't convert root to a document"))
		return n
	}

	n.write = true

	// grab path from previous
	// and delete last op
	last_op_index := len(n.ops) - 1
	name := n.ops[last_op_index].name
	n.ops = n.ops[:last_op_index]

	n.ops = append(n.ops,
		op{
			opType:  opTypeCreateDocument,
			name:    name,
			handler: opCreateDocument,
		},
	)
	return n
}

// return a copy of the Stack Error
func (n *Query) Errors() []error {
	ret := make([]error, len(n.errors))
	copy(ret, n.errors)
	return ret
}

func (n *Query) Clear() *Query {
	n.write = false
	n.ops = n.ops[:0]
	n.errors = n.errors[:0]
	return n
}

func (n *Query) Commit() error {
	n.seer.lock.Lock()
	defer n.seer.lock.Unlock()
	n.write = true
	if len(n.errors) > 0 {
		return fmt.Errorf("%d errors preventing commit.", len(n.errors))
	}

	var (
		path []string  = make([]string, 0)
		doc  *yamlNode // nil when created here
		err  error
	)
	for _, op := range n.ops {
		path, doc, err = op.handler(op, n, path, doc)
		if err != nil {
			return fmt.Errorf("Commiting failed with %s", err.Error())
		}
	}

	return nil
}

func (n *Query) Value(dst interface{}) error {
	n.seer.lock.Lock()
	defer n.seer.lock.Unlock()
	n.write = false
	if len(n.errors) > 0 {
		return fmt.Errorf("%d errors preventing getting value.", len(n.errors))
	}

	var (
		path []string  = make([]string, 0)
		doc  *yamlNode // nil when created here
		err  error
	)
	for _, op := range n.ops {
		path, doc, err = op.handler(op, n, path, doc)
		if err != nil {
			return fmt.Errorf("Value failed with %s", err.Error())
		}
	}

	if doc == nil {
		//let's see if we're looking at a folder
		_path := "/" + pathUtils.Join(path)
		if st, exist := n.seer.fs.Stat(_path); exist == nil && st.IsDir() == true {
			// it's a folder
			dirFiles, err := afero.ReadDir(n.seer.fs, _path)
			if err != nil {
				return fmt.Errorf("parsing folder `%s` failed with %w", path, err)
			}

			_dst := make([]string, 0)
			for _, f := range dirFiles {
				if f.IsDir() == true {
					_dst = append(_dst, f.Name())
				} else {
					fname := f.Name()
					item := strings.TrimSuffix(fname, ".yaml")
					if item+".yaml" == fname {
						_dst = append(_dst, item)
					}
				}
			}

			switch dst.(type) {
			case *interface{}:
				*dst.(*interface{}) = _dst
			case *[]string:
				*dst.(*[]string) = _dst
			default:
				return fmt.Errorf("value of a folder can only be mapped to `*[]string` or *interface{} not `%T`", dst)
			}

			return nil
		} else {
			return fmt.Errorf("no data found for %s", path)
		}
	}

	err = doc.this.Decode(dst)
	if err != nil {
		return fmt.Errorf("decode(%T) failed with %s", dst, err)
	}

	return nil
}

func (n *Query) List() ([]string, error) {
	var val interface{}
	err := n.Value(&val)
	if err != nil {
		return nil, fmt.Errorf("listing keys failed with %s", err)
	}

	switch val.(type) {
	case []string:
		return val.([]string), nil
	case map[string]interface{}:
		return maps.Keys(val.(map[string]interface{})), nil
	case map[interface{}]interface{}:
		return maps.Keys(maps.SafeInterfaceToStringKeys(val.(map[interface{}]interface{}))), nil
	default:
		return nil, fmt.Errorf("listing keys failed with %T is not a map or a slice", val)
	}
}

func (n *Query) dump() {
	fmt.Printf("Ops %+v\n", n.ops)
	fmt.Printf("Errors %+v\n", n.errors)
}
