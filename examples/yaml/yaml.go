package main

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func walk(sp string, t *yaml.Node) {
	fmt.Printf("%s %p | %+v | Alias: %+v\n", sp, t, t, t.Alias)
	for _, i := range t.Content {
		walk(sp+" ", i)
	}
}

const (
	DocumentNode int = 1 << iota
	SequenceNode
	MappingNode
	ScalarNode
	AliasNode
)

type yamlMap map[*yaml.Node]*yaml.Node
type yamlSeq []*yaml.Node

type yamlNode struct {
	Map map[string]yamlNode
	Seq []yamlNode
	Val yaml.Node
}

func (y yamlMap) UnmarshalYAML(value *yaml.Node) error {
	walk("> ", value)
	return nil
}

func main() {
	content := `
---
test:
  zero:
  val1: 1 #some coment
  val2: some text
  3:
   - item1
   - item2

test2: &tt cecwe 
test3: *tt
`

	a := &yamlNode{}
	err := yaml.Unmarshal([]byte(content), a)

	fmt.Printf(">>> %+v | %v\n", a, err)
	fmt.Println("------------------")

	b := &yaml.Node{}
	err = yaml.Unmarshal([]byte("test: test"), b)

	fmt.Printf("B>>> %+v | %v\n", b.Content[0].Content[1], err)

	var s string
	b.Content[0].Content[1].Decode(&s)
	fmt.Printf("B>>> %s\n", s)

	b.Content[0].Content[1].Encode(map[string]interface{}{
		"yo": 1,
		"mo": "A",
	})

	fmt.Printf("B>>> %+v | %v\n", b.Content[0].Content[1], err)

	walk("", b)

	out, _ := yaml.Marshal(b)
	fmt.Println("OUT:\n", string(out))

	fmt.Println("------------------")

	t := &yaml.Node{}
	yaml.Unmarshal([]byte(content), t)

	walk("", t)

	out, _ = yaml.Marshal(t)
	fmt.Println("OUT:", string(out))

}
