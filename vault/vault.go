package vault

import (
	"fmt"
	"os"

	"github.com/erikgeiser/promptkit/textinput"
	"github.com/tobischo/gokeepasslib/v3"
)

type Database *gokeepasslib.Database

func CreateNewKeepassDatabase(path, fileName, masterPassword, groupName string) error {
	db := gokeepasslib.NewDatabase()
	db.Content.Meta.DatabaseName = groupName
	db.Credentials = gokeepasslib.NewPasswordCredentials(masterPassword)

	// Add a new group with the provided name
	rootGroup := gokeepasslib.Group{Name: groupName}
	db.Content.Root.Groups = append(db.Content.Root.Groups, rootGroup)

	// Lock entries using stream cipher
	if err := db.LockProtectedEntries(); err != nil {
		return err
	}

	// Write the database to a new file
	filePath := path + "/" + fileName
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	keepassEncoder := gokeepasslib.NewEncoder(file)
	err = keepassEncoder.Encode(db)
	if err != nil {
		return err
	}
	return nil
}

func OpenKeepassDatabase(dbPath, masterPassword string) (*gokeepasslib.Database, error) {
	file, err := os.Open(dbPath)
	if err != nil {
		return nil, err
	}

	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(masterPassword)
	err = gokeepasslib.NewDecoder(file).Decode(db)
	if err != nil {
		return nil, err
	}

	if err := db.UnlockProtectedEntries(); err != nil {
		return nil, err
	}

	return db, nil
}

//		if top-level group, pass in nil parentGroup and append new group to db.Content.Root.Groups
//	 	if sub-group, pass in parentGroup and append new group to parentGroup.Groups
func NewGroup(db *gokeepasslib.Database, parentGroup *gokeepasslib.Group, groupName string) {
	var targetGroup *gokeepasslib.Group
	// top-level group creation
	if parentGroup == nil {
		for _, group := range db.Content.Root.Groups {
			if group.Name == groupName {
				targetGroup = &group
				break
			}
		}

		if targetGroup == nil {
			targetGroup = &gokeepasslib.Group{
				Name: groupName,
			}
			db.Content.Root.Groups = append(db.Content.Root.Groups, *targetGroup)
		}
		return
	}
	// sub-group creation
	newGroup := &gokeepasslib.Group{
		Name: groupName,
	}
	parentGroup.Groups = append(parentGroup.Groups, *newGroup)
}

func GetGroup(db *gokeepasslib.Database, groupName string) *gokeepasslib.Group {
	for _, group := range db.Content.Root.Groups {
		if group.Name == groupName {
			return &group
		}
	}
	return nil
}

func GetGroupEntry(group *gokeepasslib.Group, title string) *gokeepasslib.Entry {
	for _, entry := range group.Entries {
		if entry.Get("Title") == title {
			return &entry
		}
	}
	return nil
}

func SaveGroupEntry(group *gokeepasslib.Group, title, username, password string) {
	newEntry := gokeepasslib.Entry{
		Values: gokeepasslib.Values{
			gokeepasslib.ValueData{Key: "Title", Value: title},
			gokeepasslib.ValueData{Key: "Username", Value: username},
			gokeepasslib.ValueData{Key: "Password", Value: password},
		},
	}
	group.Entries = append(group.Entries, newEntry)
}

func CloseDB(db *gokeepasslib.Database) error {
	if db != nil {
		if err := db.LockProtectedEntries(); err != nil {
			return err
		}
		db = nil
	}
	return nil
}

func PromptDBPath() (string, error) {
	pathPrompt := textinput.New("Path to vault database: ")
	pathPrompt.Placeholder = "path cannot be empty"

	dbPath, err := pathPrompt.RunPrompt()
	if err != nil {
		return "", "", err
	}
	return dbPath, nil
}

func PromptEntryCredentials() (string, string, error) {
	userPrompt := textinput.New("Username: ")
	userPrompt.Placeholder = "Enter username"

	username, err := userPrompt.RunPrompt()
	if err != nil {
		return "", "", err
	}

	passPrompt := textinput.New("Password:")
	passPrompt.Placeholder = "Enter password"
	passPrompt.Validate = func(s string) error {
		if len(s) < 1 {
			return fmt.Errorf("password cannot be empty")
		}

		return nil
	}
	passPrompt.Hidden = true
	passPrompt.Template += `
	{{- if .ValidationError -}}
		{{- print " " (Foreground "1" .ValidationError.Error) -}}
	{{- end -}}`

	password, err := passPrompt.RunPrompt()
	if err != nil {
		return "", "", err
	}

	return username, password, nil
}

func DbUnlockPrompt() (string, error) {
	passPrompt := textinput.New("Enter the Keepass database password:")
	passPrompt.Placeholder = "Enter password"
	passPrompt.Validate = func(s string) error {
		if len(s) < 1 {
			return fmt.Errorf("password cannot be empty")
		}

		return nil
	}
	passPrompt.Hidden = true
	passPrompt.Template += `
			{{- if .ValidationError -}}
				{{- print " " (Foreground "1" .ValidationError.Error) -}}
			{{- end -}}`

	password, err := passPrompt.RunPrompt()
	if err != nil {
		return "", err
	}

	return password, nil
}
