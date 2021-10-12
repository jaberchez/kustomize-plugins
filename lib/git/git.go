package git

import (
	"fmt"

	"gopkg.in/ini.v1"
)

const (
	// GenericRegex to find git lines
	//
	// Notes: Is posible to usar modifiers with |
	//        Available modifiers (they can be cocatenated | base64 | indent4):
	//          - base64
	//          - indent4
	GenericRegex string = `\${\s*git:(.+?)\s*(\|.+?)?}`

	// SpecificRegex to find specific git data
	// Example: ${ git:LDAP_BIND_PASSWORD | base64 }
	SpecificRegex string = `\${\s*git:%s\s*(\|.+?)?}`
)

// LoadFileConf load git file configuration
func LoadFileConf(fileConf string) (*ini.File, error) {
	// Load git file configuration
	gitConf, err := ini.Load(fileConf)

	if err != nil {
		return nil, fmt.Errorf("Fail to read file: %v", err)
	}

	return gitConf, nil
}
