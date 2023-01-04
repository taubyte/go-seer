package seer

import (
	"testing"

	slices "github.com/taubyte/utils/slices/string"
)

func TestList(t *testing.T) {
	seer, err := New(fixtureFS(true, "/"))
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("Basic empty", func(t *testing.T) {
		funcs := []interface{ List() ([]string, error) }{
			seer,
			seer.Get("parent"),
			seer.Get("parent").Get("p").Document(),
			seer.Get("a").Get("b").Get("C").Document().Get("a").Get("orange"),
		}
		for _, f := range funcs {
			listItems, _ := f.List()
			if len(listItems) != 0 {
				t.Error("list should be empty")
				break
			}
		}
	})
	t.Run("Basic set and list", func(t *testing.T) {
		seer.Get("parent").Get("p").Commit()
		listItems, err := seer.List()
		if err != nil {
			t.Errorf("list failed with error: %s", err.Error())
			return
		}
		if listItems[0] != "parent" {
			t.Error("expected parent")
			return
		}

		listItems, err = seer.Get("parent").List()
		if err != nil {
			t.Errorf("list failed with error: %s", err.Error())
			return
		}
		if listItems[0] != "p" {
			t.Error("expected p")
			return
		}

	})

	t.Run("Basic multi-set run", func(t *testing.T) {
		listItems, err := seer.Get("parent").List()
		if listItems[0] != "p" || err != nil {
			t.Error("expected p")
			return
		}

		items := []string{"oranges", "bananas", "pears", "pineapples", "coconuts"}
		for _, i := range items {
			seer.Get("parent").Get(i).Commit()
		}
		listItems, err = seer.Get("parent").List()
		if err != nil {
			t.Errorf("list failed with error: %s", err.Error())
		}

		for _, i := range items {
			if inSlice(listItems, i) == false {
				t.Errorf("%s not found in %s", i, listItems)
			}
		}

		t.Run("Test proof", func(t *testing.T) {
			items := []string{"oranges", "bananas", "pears", "pineapples", "coconuts"}
			newItems := append(items, "nanacoco")

			var foundErr bool
			for _, i := range newItems {
				if inSlice(items, i) == false {
					foundErr = true
				}
			}
			if foundErr == false {
				t.Error("Test doesn't work")
			}
		})
	})

	t.Run("Deep multi-set run", func(t *testing.T) {
		items := []string{"oranges", "bananas", "pears", "pineapples", "coconuts"}
		for _, i := range items {
			seer.Get("parent").Get("sad").Get("fruits").Get(i).Commit()
		}
		listItems, err := seer.Get("parent").Get("sad").Get("fruits").List()
		if err != nil {
			t.Errorf("list failed with error: %s", err.Error())
		}

		for _, i := range items {
			if inSlice(listItems, i) == false {
				t.Errorf("%s not found in %s", i, listItems)
			}
		}
	})

	t.Run("Deep multi-set run with commit and delete", func(t *testing.T) {
		query := seer.Get("parent").Get("sad").Get("fruits")

		items := []string{"oranges", "bananas", "pears", "pineapples", "coconuts"}
		for _, i := range items {
			query.Fork().Get(i).Commit()
		}
		toDelete := []string{"bananas", "pears"}
		for _, i := range toDelete {
			query.Fork().Get(i).Delete().Commit()
		}

		expectedItems := []string{"oranges", "pineapples", "coconuts"}
		listItems, _ := query.List()
		for _, i := range expectedItems {
			if inSlice(listItems, i) == false {
				t.Errorf("%s not found in %s", i, listItems)
			}
		}

		for _, i := range toDelete {
			if inSlice(listItems, i) == true {
				t.Errorf("%s found in %s", i, listItems)
			}
		}
	})

	t.Run("listing on a document", func(t *testing.T) {
		documentName := "some-doc"
		listItem1 := "pears"
		listItem2 := "bananas"

		err = seer.Get(documentName).Document().Get(listItem1).Set(10).Commit()
		if err != nil {
			t.Error(err)
			return
		}

		err = seer.Get(documentName).Document().Get(listItem2).Set(20).Commit()
		if err != nil {
			t.Error(err)
			return
		}

		val, err := seer.Get(documentName).List()
		if err != nil {
			t.Error(err)
			return
		}

		if slices.Contains(val, listItem1) == false {
			t.Errorf("%s not found in `%v`", listItem1, val)
			return
		}
		if slices.Contains(val, listItem2) == false {
			t.Errorf("%s not found in `%v`", listItem2, val)
			return
		}
	})
}
