package vault

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/vault/api"
)

const (
	// GenericRegex to find vault lines
	// ${vault:path-secret@key}
	// Notes: Is posible to usar modifiers with |
	//        Available modifiers (they can be cocatenated | base64 | indent4):
	//          - base64
	//          - indent4
	// Example: ${ vault:test/data/sync-ldap@bindPassword | base64 }
	GenericRegex string = `\${\s*vault:(.+?)@(.+?)\s*(\|.+?)?}`

	// SpecificRegex to find specific vault data
	SpecificRegex string = `\${\s*vault:%s@%s\s*(\|.+?)?}`
)

// Configuration is the configuration to access Vault
type Configuration struct {
	VaultHost  string
	VaultToken string
}

// NotFoundError is a custorm error for not found secret or key
type NotFoundError struct {
	Msg string
}

func (e *NotFoundError) Error() string {
	return e.Msg
}

// GetSecret get secret from Vault
func (c *Configuration) GetSecret(pathSecret string, key string) (string, error) {
	client, err := createVaultClient(c.VaultHost, c.VaultToken)

	if err != nil {
		return "", err
	}

	vaultData, err := client.Logical().Read(pathSecret)

	if err != nil {
		return "", err
	}

	if vaultData == nil {
		// Secret does not exist
		return "", &NotFoundError{fmt.Sprintf("Secret \"%s\" not found", pathSecret)}
	}

	v := vaultData.Data["data"]

	if v == nil {
		return "", &NotFoundError{fmt.Sprintf("Data not found in secret \"%s\"", pathSecret)}
	}

	d := v.(map[string]interface{})

	for k, v := range d {
		if k == key {
			return v.(string), nil
		}
	}

	return "", &NotFoundError{fmt.Sprintf("Key \"%s\" not found", key)}
}

func createVaultClient(vaultHost, vaultToken string) (*api.Client, error) {
	var httpClient = &http.Client{Timeout: 10 * time.Second}

	client, err := api.NewClient(&api.Config{Address: vaultHost, HttpClient: httpClient})

	if err != nil {
		return nil, err
	}

	client.SetToken(vaultToken)

	return client, nil
}
