package dao

import (
	"database/sql"
	"time"

	"github.com/gitKashish/kosh/src/internals/logger"
	"github.com/gitKashish/kosh/src/internals/model"
)

func GetCredentialByLabelAndUser(label, user string) (*model.Credential, error) {
	var credential model.Credential
	stmt, err := db.Prepare(`
		SELECT label, user, secret, ephemeral, nonce FROM credentials
		WHERE label = ? AND user = ?
	`)
	if err != nil {
		logger.Error("failed to prepare statement to fetch credential info")
		return nil, err
	}

	err = stmt.QueryRow(label, user).Scan(&credential.Label, &credential.User, &credential.Secret, &credential.Ephemeral, &credential.Nonce)

	if err == sql.ErrNoRows {
		return nil, err
	}

	if err != nil {
		logger.Error("unable to fetch credential")
		return nil, err
	}

	return &credential, nil
}

func AddCredential(credential *model.Credential) error {
	query := `
		INSERT INTO credentials (label, user, secret, ephemeral, nonce)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (label, user)
		DO UPDATE SET
			secret = excluded.secret,
			ephemeral = excluded.ephemeral,
			nonce = excluded.nonce
	`

	stmt, err := db.Prepare(query)
	if err != nil {
		logger.Error("error preparing statement")
		return err
	}

	_, err = stmt.Exec(credential.Label, credential.User, credential.Secret, credential.Ephemeral, credential.Nonce)
	if err != nil {
		logger.Error("error inserting credential")
	}

	return nil
}

func GetCredentialsByUserOrLabel(label, user string) ([]model.CredentialSummary, error) {
	query := `
		SELECT id, label, user, created_at, updated_at FROM credentials
		WHERE TRUE 
	`

	params := []any{}

	if label != "" {
		query = query + ` AND label LIKE ? `
		params = append(params, "%"+label+"%")
	}

	if user != "" {
		query = query + ` AND user LIKE ? `
		params = append(params, "%"+user+"%")
	}

	rows, err := db.Query(query, params...)
	if err != nil {
		logger.Debug("failed to fetch list of saved credentials")
		return nil, err
	}
	defer rows.Close()

	credentials := []model.CredentialSummary{}
	for rows.Next() {
		var credential model.CredentialSummary
		var createdAtStr, updatedAtStr string

		if err := rows.Scan(
			&credential.Id,
			&credential.Label,
			&credential.User,
			&createdAtStr,
			&updatedAtStr,
		); err != nil {
			logger.Debug("unable to scan row")
			return nil, err
		}

		credential.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			logger.Debug("unable to parse created at time: %s", createdAtStr)
			return nil, err
		}

		credential.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
		if err != nil {
			logger.Debug("unable to parse updated at time: %s", updatedAtStr)
			return nil, err
		}

		credentials = append(credentials, credential)
	}

	if rows.Err() != nil {
		logger.Debug("error iterating over rows")
		return nil, err
	}

	return credentials, nil
}
