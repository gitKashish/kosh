package dao

import (
	"database/sql"
	"fmt"

	"github.com/gitKashish/kosh/src/internals/model"
)

func GetCredentialByLabelAndUser(label, user string) *model.Credential {
	var credential model.Credential
	stmt, err := db.Prepare(`
		SELECT label, user, secret, ephemeral, nonce FROM credentials
		WHERE label = ? AND user = ?
	`)
	if err != nil {
		fmt.Println("[Error] failed to prepare statement to fetch credential info")
		fmt.Printf("[Debug] %s\n", err.Error())
		return nil
	}

	err = stmt.QueryRow(label, user).Scan(&credential.Label, &credential.User, &credential.Secret, &credential.Ephemeral, &credential.Nonce)

	if err == sql.ErrNoRows {
		fmt.Println("[Error] no matching credential found")
		return nil
	}

	if err != nil {
		fmt.Println("[Error] unable to fetch credential")
		fmt.Printf("[Debug] %s\n", err.Error())
		return nil
	}

	return &credential
}

func AddCredential(credential *model.Credential) error {
	query := `
		INSERT INTO credentials (label, user, secret, ephemeral, nonce)
		VALUES (?, ?, ?, ?, ?)
	`

	stmt, err := db.Prepare(query)
	if err != nil {
		fmt.Println("[Error] error preparing statement")
		return err
	}

	_, err = stmt.Exec(credential.Label, credential.User, credential.Secret, credential.Ephemeral, credential.Nonce)
	if err != nil {
		fmt.Println("[Error] error inserting credential")
	}

	return nil
}
