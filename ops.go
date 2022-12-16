package seer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	pathUtils "github.com/taubyte/utils/path"
)

func opDelete(this op, query *Query, path []string, value *yamlNode) ([]string, *yamlNode, error) {
	if query.write == false {
		return path, nil, errors.New("Failed to call Delete() during a read query")
	}

	if value == nil || value.parent == nil {
		return _opDeleteInFileSystem(this, query, path, nil)
	} else {
		// else we're dealing with inside
		return _opDeleteInYaml(this, query, path, value)
	}
}

func _opDeleteInYaml(this op, query *Query, path []string, value *yamlNode) ([]string, *yamlNode, error) {

	if query.write == false {
		return path, nil, errors.New("Failed to call Delete() during a read query")
	}

	if value == nil || value.parent == nil || value.prev == nil {
		return path, nil, errors.New("Failed to call Delete() outside a value or document")
	}

	parentNodeContent := value.parent.Content
	value.parent.Content = make([]*yaml.Node, 0)

	for _, elm := range parentNodeContent {
		if elm == value.prev || elm == value.this {
			continue
		}
		value.parent.Content = append(value.parent.Content, elm)
	}

	return path, &yamlNode{parent: value.parent, prev: nil, this: nil}, nil
}

func _opDeleteInFileSystem(this op, query *Query, _path []string, value *yamlNode) ([]string, *yamlNode, error) {
	_path = append(_path, this.name)
	path := "/" + pathUtils.Join(_path)

	st, err := query.seer.fs.Stat(path)
	if err != nil {

		// now we know it's a file, it sure is not a yaml file by our standards
		return _path, nil, fmt.Errorf("Unsupported file `%s`. Is this a Document?", path)

	}

	if st.IsDir() {
		// it's a dir => nothing to be done
		for k, _ := range query.seer.documents {
			if strings.HasPrefix(k, path) {
				delete(query.seer.documents, k)
			}
		}
		err := query.seer.fs.RemoveAll(path)
		return _path, nil, err
	}
	// let's cleanup
	_, exists := query.seer.documents[path]
	if exists == true {
		// we know it is a file
		delete(query.seer.documents, path)
	}
	err = query.seer.fs.Remove(path)
	return _path, nil, err

}

func opSetInYaml(this op, query *Query, path []string, value *yamlNode) ([]string, *yamlNode, error) {

	if query.write == false {
		return path, nil, errors.New("Failed to call Set() during a read query")
	}

	if value == nil || value.this == nil {
		return path, nil, errors.New("Failed to call Set() outside a document")
	}

	parentNode := value.parent
	curNode := value.this

	curNode_HeadComment := curNode.HeadComment
	curNode_LineComment := curNode.LineComment
	curNode_FootComment := curNode.FootComment

	err := curNode.Encode(this.value)

	curNode.HeadComment = curNode_HeadComment
	curNode.LineComment = curNode_LineComment
	curNode.FootComment = curNode_FootComment

	return path, &yamlNode{parent: parentNode, prev: value.prev, this: curNode}, err
}

func opGetOrCreate(this op, query *Query, path []string, value *yamlNode) ([]string, *yamlNode, error) {
	if value == nil {
		if query.write == true {
			return _opGetOrCreateInFileSystem(this, query, path, nil)
		} else {
			return _opGetInFileSystem(this, query, path, nil)
		}
	} else {
		// else we're dealing with inside
		return _opGetInYaml(this, query, path, value)
	}
}

func _opGetInYaml(this op, query *Query, path []string, value *yamlNode) ([]string, *yamlNode, error) {

	if value == nil || value.this == nil {
		return path, nil, fmt.Errorf("Can not find %s in the empty document %s", this.name, pathUtils.Join(path))
	}

	path = append(path, this.name)
	parentNode := value.parent
	curNode := value.this
	if curNode.Kind == yaml.DocumentNode {
		if len(curNode.Content) != 1 {
			return path, nil, fmt.Errorf("Failed to process empty document at %s", pathUtils.Join(path))
		}
		parentNode = curNode
		curNode = curNode.Content[0]
	}
	if curNode.Kind == yaml.MappingNode {
		parentNode = curNode
		for i := 0; i+1 < len(curNode.Content); i += 2 {
			if curNode.Content[i].Kind == yaml.ScalarNode && curNode.Content[i].Value == this.name {
				// we got it
				return path, &yamlNode{parent: parentNode, prev: curNode.Content[i], this: curNode.Content[i+1]}, nil
			}
		}

		if query.write == true {
			parentNode = curNode
			curNode = &yaml.Node{}
			curNode.Encode(map[string]interface{}{this.name: nil})
			parentNode.Content = append(parentNode.Content, curNode.Content...)
			return path, &yamlNode{parent: parentNode, prev: curNode.Content[0], this: curNode.Content[1]}, nil
		}
		// else, we return error
		return path, nil, fmt.Errorf("Can not find %s", pathUtils.Join(path))

	}
	if curNode.Kind == yaml.SequenceNode {
		_idx, err := strconv.ParseInt(this.name, 0, 32)
		if err != nil {
			return path, nil, fmt.Errorf("Failed to process index %s. Error: %s", this.name, err.Error())
		}
		_index := int(_idx)
		if _index >= len(curNode.Content) {
			if query.write == true {
				parentNode = curNode
				curNode = &yaml.Node{}
				curNode.Encode(nil)
				parentNode.Content = append(parentNode.Content, curNode)
				return path, &yamlNode{parent: parentNode, prev: nil, this: curNode}, nil
			} else {
				return path, nil, fmt.Errorf("Index %d out of range (Length: %d)", _index, len(curNode.Content))
			}
		}

		return path, &yamlNode{parent: parentNode, prev: nil, this: curNode.Content[_index]}, nil
	}

	if query.write == true {

		curNode.Encode(map[string]interface{}{this.name: nil})
		return path, &yamlNode{parent: parentNode, prev: curNode, this: curNode.Content[1]}, nil
	}
	//else

	return path, nil, fmt.Errorf("Can not find %s", pathUtils.Join(path))
}

func _opGetOrCreateInFileSystem(this op, query *Query, _path []string, value *yamlNode) ([]string, *yamlNode, error) {
	_path = append(_path, this.name)
	path := "/" + pathUtils.Join(_path)
	doc, exists := query.seer.documents[path+".yaml"]
	if exists == true {
		_path[len(_path)-1] += ".yaml"
		return _path, &yamlNode{parent: nil, this: doc}, nil
	}
	st, err := query.seer.fs.Stat(path)
	if err != nil {
		// let's check if we're not looking for a yaml file first
		st, err = query.seer.fs.Stat(path + ".yaml")
		if err != nil {
			// we assume that the folder does not exit and we create
			err = query.seer.fs.Mkdir(path, 0750)
			if err != nil {
				return _path, nil, fmt.Errorf("Creating directory %s failed with %s", path, err.Error())
			}
			return _path, nil, nil
		} else if st.IsDir() {
			return _path, nil, fmt.Errorf("Not allowed directory `%s.yaml`", path)
		}

		// it's a yaml file
		doc, err := query.seer.loadYamlDocument(path + ".yaml")
		_path[len(_path)-1] += ".yaml"
		return _path, &yamlNode{parent: nil, this: doc}, err

	}
	if st.IsDir() {
		// it's a dir => nothing to be done
		return _path, nil, nil
	}
	// now we know it's a file, it sure is not a yaml file by our standards
	return _path, nil, fmt.Errorf("Unsupported file `%s`", path)
}

func _opGetInFileSystem(this op, query *Query, _path []string, value *yamlNode) ([]string, *yamlNode, error) {
	_path = append(_path, this.name)
	path := "/" + pathUtils.Join(_path)
	doc, exists := query.seer.documents[path+".yaml"]
	if exists == true {
		_path[len(_path)-1] += ".yaml"
		return _path, &yamlNode{parent: nil, this: doc}, nil
	}
	st, err := query.seer.fs.Stat(path)
	if err != nil {
		// let's check if we're not looking for a yaml file first
		st, err = query.seer.fs.Stat(path + ".yaml")
		if err != nil {
			// the folder does not exit
			return _path, nil, fmt.Errorf("Fetching %s failed with %s", path, err.Error())
		} else if st.IsDir() {
			return _path, nil, fmt.Errorf("Not allowed directory `%s.yaml`", path)
		}

		// it's a yaml file
		doc, err := query.seer.loadYamlDocument(path + ".yaml")
		_path[len(_path)-1] += ".yaml"
		return _path, &yamlNode{parent: nil, this: doc}, err

	}
	if st.IsDir() {
		// it's a dir => nothing to be done
		return _path, nil, nil
	}
	// now we know it's a file, it sure is not a yaml file by our standards
	return _path, nil, fmt.Errorf("Unsupported file `%s`", path)
}

func opCreateDocument(this op, query *Query, _path []string, value *yamlNode) ([]string, *yamlNode, error) {
	_path = append(_path, this.name+".yaml")
	path := "/" + pathUtils.Join(_path)
	// Check for it first

	doc, exists := query.seer.documents[path]
	if exists == true {
		return _path, &yamlNode{parent: nil, this: doc}, nil
	}

	st, err := query.seer.fs.Stat(path)
	if err == nil {
		if st.IsDir() == true {
			return _path, nil, fmt.Errorf("Can't create document: `%s` is a directory", path)
		}
	} else { // we need to create
		if query.write == true {
			f, err := query.seer.fs.Create(path)
			if err != nil {
				return _path, nil, fmt.Errorf("Creating yaml file %s failed with %s", path, err.Error())
			}
			defer f.Close()
			f.WriteString("----\n") // we need to write something so yaml package does not fail
		} else {
			return _path, nil, fmt.Errorf("Document: `%s` does not exist", path)
		}

	}

	doc, err = query.seer.loadYamlDocument(path)
	return _path, &yamlNode{parent: nil, this: doc}, err
}
