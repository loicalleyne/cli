package vault

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pterm/pterm"
	"github.com/tobischo/gokeepasslib/v3"
)

func ListEntries(g *gokeepasslib.Group) []string {
	entries := make([]string, 0)

	for _, entry := range g.Entries {
		entries = append(entries, entry.GetTitle())
	}

	for _, group := range g.Groups {
		subEntries := ListEntries(&group)
		for i, val := range subEntries {
			subEntries[i] = fmt.Sprintf("%s/%s", group.Name, val)
		}

		entries = append(entries, subEntries...)
	}

	return entries
}

func ReadEntry(selection string, g *gokeepasslib.Group) (*gokeepasslib.Entry, error) {
	for i, entry := range g.Entries {
		if entry.GetTitle() == selection {
			return &g.Entries[i], nil
		}
	}

	selectors := strings.Split(selection, "/")

	if len(selectors) == 1 {
		for i, entry := range g.Entries {
			if entry.GetTitle() == selectors[0] {
				return &g.Entries[i], nil
			}
		}
	} else {
		for i, group := range g.Groups {
			if group.Name == selectors[0] {
				return ReadEntry(strings.Join(selectors[1:], "/"), &g.Groups[i])
			}
		}
	}

	entries := SearchEntries(selectors, g)
	if len(entries) < 1 {
		return nil, fmt.Errorf("No entry found")
	}
	selection, _ = pterm.DefaultInteractiveSelect.WithMaxHeight(40).WithOptions(entries).WithDefaultText("Select entry").Show()

	index, err := strconv.Atoi(selection)
	if err != nil {
		return nil, err
	}

	return ReadEntry(entries[index], g)
}

func SearchEntries(selectors []string, g *gokeepasslib.Group) []string {
	entries := ListEntries(g)

	selector := strings.ToLower(strings.Join(selectors, "/"))

	selectedEntries := make([]string, 0)

	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry), selector) {
			selectedEntries = append(selectedEntries, entry)
		}
	}

	return selectedEntries
}
