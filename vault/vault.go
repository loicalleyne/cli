package vault

import (
	"fmt"
	"os"

	"go.uber.org/atomic"

	"github.com/erikgeiser/promptkit/textinput"
	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

type (
	Database  *gokeepasslib.Database
	VaultInfo struct {
		Unlocked       atomic.Bool
		DBPath         string
		DBName         string
		DBFileName     string
		MasterPassword string
	}
)

func CreateNewKeepassDatabase(v *VaultInfo, groupName string) error {
	db := gokeepasslib.NewDatabase(gokeepasslib.WithDatabaseKDBXVersion4())

	db.Credentials = gokeepasslib.NewPasswordCredentials(v.MasterPassword)
	db.Content.Meta.DatabaseName = v.DBName
	db.Content.Root = gokeepasslib.NewRootData()

	// Add a new group with the provided name
	newGroup := gokeepasslib.NewGroup()
	newGroup.Name = groupName
	db.Content.Root.Groups[0] = newGroup

	// Lock entries using stream cipher
	if err := db.LockProtectedEntries(); err != nil {
		return err
	}

	// Write the database to a new file
	filePath := v.DBPath + "/" + v.DBFileName
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

func OpenKeepassDatabase(v *VaultInfo) (*gokeepasslib.Database, error) {
	file, err := os.Open(v.DBPath + "/" + v.DBFileName)
	if err != nil {
		return nil, err
	}

	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(v.MasterPassword)
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
func NewGroup(db *gokeepasslib.Database, v *VaultInfo, parentGroup *gokeepasslib.Group, groupName string) error {
	var targetGroup *gokeepasslib.Group
	// top-level group creation
	if parentGroup == nil {
		for i, group := range db.Content.Root.Groups[0].Groups {
			if group.Name == groupName {
				targetGroup = &db.Content.Root.Groups[0].Groups[i]
				break
			}
		}
		// sub-group of root
		if targetGroup == nil {
			g := gokeepasslib.NewGroup()
			g.Name = groupName
			db.Content.Root.Groups[0].Groups = append(db.Content.Root.Groups[0].Groups, g)
		}
		err := SaveDB(db, v)
		if err != nil {
			return err
		}
		return nil
	}
	// sub-group creation
	newGroup := gokeepasslib.NewGroup()
	newGroup.Name = groupName
	parentGroup.Groups = append(parentGroup.Groups, newGroup)
	err := SaveDB(db, v)
	if err != nil {
		return err
	}
	return nil
}

func GetGroup(db *gokeepasslib.Database, groupName string) *gokeepasslib.Group {
	if len(db.Content.Root.Groups) > 0 && db.Content.Root.Groups[0].Name == groupName {
		return &db.Content.Root.Groups[0]
	}
	if len(db.Content.Root.Groups) == 0 {
		return nil
	}
	rootGroup := db.Content.Root.Groups[0]
	for i, group := range rootGroup.Groups {
		if group.Name == groupName {
			return &db.Content.Root.Groups[0].Groups[i]
		}
	}
	return nil
}

func GetParentGroupsIndex(db *gokeepasslib.Database, groupName string) (*gokeepasslib.Group, int) {
	if len(db.Content.Root.Groups) > 0 && db.Content.Root.Groups[0].Name == groupName {
		return &db.Content.Root.Groups[0], 0
	}
	if len(db.Content.Root.Groups) == 0 {
		return nil, 0
	}
	rootGroup := db.Content.Root.Groups[0]
	for i, group := range rootGroup.Groups {
		if group.Name == groupName {
			return &group, i
		}
	}
	return nil, 0
}

func GetGroupEntry(group *gokeepasslib.Group, title string) *gokeepasslib.Entry {
	for i, entry := range group.Entries {
		if entry.GetTitle() == title {
			return &group.Entries[i]
		}
	}
	return nil
}

func SaveGroupEntry(db *gokeepasslib.Database, group *gokeepasslib.Group, v *VaultInfo, title, username, password string) error {
	newEntry := gokeepasslib.NewEntry()
	vTitle := toValueData("Title", title)
	vUser := toValueData("Username", username)
	vPass := toProtectedValueData("Password", password)
	newEntry.Values = append(newEntry.Values, vTitle)
	newEntry.Values = append(newEntry.Values, vUser)
	newEntry.Values = append(newEntry.Values, vPass)
	group.Entries = append(group.Entries, newEntry)

	err := SaveDB(db, v)
	if err != nil {
		return err
	}

	return nil
}

func toValueData(key, value string) gokeepasslib.ValueData {
	return gokeepasslib.ValueData{Key: key, Value: gokeepasslib.V{Content: value}}
}

func toProtectedValueData(key, value string) gokeepasslib.ValueData {
	return gokeepasslib.ValueData{
		Key:   key,
		Value: gokeepasslib.V{Content: value, Protected: w.NewBoolWrapper(true)},
	}
}

func SaveDB(db *gokeepasslib.Database, v *VaultInfo) error {
	if err := db.LockProtectedEntries(); err != nil {
		return err
	}
	file, err := os.OpenFile(v.DBPath+"/"+v.DBFileName, os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	keepassEncoder := gokeepasslib.NewEncoder(file)
	if err := keepassEncoder.Encode(db); err != nil {
		file.Close()
		return err
	}
	file.Close()

	newdb, err := OpenKeepassDatabase(v)
	if err != nil {
		return err
	}
	db = newdb
	return nil
}

func CloseDB(db *gokeepasslib.Database, v *VaultInfo) error {
	if db != nil {
		if err := db.LockProtectedEntries(); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(v.DBPath+"/"+v.DBFileName, os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	keepassEncoder := gokeepasslib.NewEncoder(file)
	if err := keepassEncoder.Encode(db); err != nil {
		return err
	}
	db = nil
	return nil
}

func PromptDBPath() (string, error) {
	pathPrompt := textinput.New("Path to vault database: ")
	pathPrompt.Placeholder = "path cannot be empty"

	dbPath, err := pathPrompt.RunPrompt()
	if err != nil {
		return "", err
	}
	return dbPath, nil
}

func PromptEntryCredentials(titleRequired bool, titleOverride string) (string, string, string, error) {
	var (
		titlePrompt *textinput.TextInput
		title       string
		err         error
	)
	if titleRequired {
		titlePrompt = textinput.New("Title: ")
		titlePrompt.Placeholder = "Enter username"

		title, err = titlePrompt.RunPrompt()
		if err != nil {
			return "", "", "", err
		}
	}
	userPrompt := textinput.New("Username: ")
	userPrompt.Placeholder = "Enter username"

	username, err := userPrompt.RunPrompt()
	if err != nil {
		return "", "", "", err
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
		return "", "", "", err
	}

	confirmPassPrompt := textinput.New("Confirm Password:")
	confirmPassPrompt.Placeholder = "Enter password"
	confirmPassPrompt.Validate = func(s string) error {
		if len(s) < 1 {
			return fmt.Errorf("password cannot be empty")
		}
		if s != password {
			return fmt.Errorf("passwords must match")
		}
		return nil
	}
	confirmPassPrompt.Hidden = true
	confirmPassPrompt.Template += `
	{{- if .ValidationError -}}
		{{- print " " (Foreground "1" .ValidationError.Error) -}}
	{{- end -}}`

	confirmPassword, err := confirmPassPrompt.RunPrompt()
	if err != nil {
		return "", "", "", err
	}
	if password != confirmPassword {
		return "", "", "", fmt.Errorf("password and confirmation do not match")
	}

	return title, username, password, nil
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
