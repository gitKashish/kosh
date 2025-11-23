package dao

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gitKashish/kosh/src/internals/constants"
	"github.com/gitKashish/kosh/src/internals/logger"
	"github.com/gitKashish/kosh/src/internals/model"
)

func GetCredentialById(id int) (*model.Credential, error) {
	var credential model.Credential
	query := `
		SELECT id, label, user, secret, ephemeral, nonce FROM credentials
		WHERE id = ?
	`

	err := db.QueryRow(query, id).Scan(&credential.Id, &credential.Label, &credential.User, &credential.Secret, &credential.Ephemeral, &credential.Nonce)

	if err == sql.ErrNoRows {
		logger.Debug("no matching credential found")
		return nil, err
	}

	if err != nil {
		logger.Error("unable to fetch credential")
		return nil, err
	}

	return &credential, nil
}

func GetCredentialByLabelAndUser(label, user string) (*model.Credential, error) {
	var credential model.Credential
	query := `
		SELECT id, label, user, secret, ephemeral, nonce FROM credentials
		WHERE label = ? AND user = ?
	`

	err := db.QueryRow(query, label, user).Scan(&credential.Id, &credential.Label, &credential.User, &credential.Secret, &credential.Ephemeral, &credential.Nonce)

	if err == sql.ErrNoRows {
		logger.Debug("no matching credential found")
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

func SearchCredentialByLabelOrUser(label, user string) ([]model.CredentialSummary, error) {
	query := `
		SELECT id, label, user, access_count, created_at, updated_at, accessed_at FROM credentials
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
		var createdAtStr, updatedAtStr, accessedAtStr string

		if err := rows.Scan(
			&credential.Id,
			&credential.Label,
			&credential.User,
			&credential.AccessCount,
			&createdAtStr,
			&updatedAtStr,
			&accessedAtStr,
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

		credential.AccessedAt, err = time.Parse(time.RFC3339, accessedAtStr)
		if err != nil {
			logger.Debug("unable to parse updated at time: %s", accessedAtStr)
			return nil, err
		}

		credentials = append(credentials, credential)
	}

	if rows.Err() != nil {
		logger.Debug("error iterating over rows")
		return nil, rows.Err()
	}

	return credentials, nil
}

// DeleteCredentialById deletes a stored credential by its ID, returns error ID is invalid
func DeleteCredentialById(id int) error {
	query := `DELETE FROM credentials WHERE id = ?`
	result, err := db.Exec(query, id)
	if affectedRows, _ := result.RowsAffected(); affectedRows != 1 {
		logger.Error("invalid credential id %d", id)
		return fmt.Errorf("no rows affected")
	}
	if err != nil {
		logger.Debug("unable to delete credential")
		return err
	}
	return nil
}

func GetAllCredentials() ([]model.Credential, error) {
	query := `SELECT id, label, user, access_count, secret, ephemeral, nonce, accessed_at FROM credentials`
	rows, err := db.Query(query)
	if err != nil {
		logger.Debug("error fetching all credentials from database")
		return nil, err
	}
	defer rows.Close()

	var credentials []model.Credential
	for rows.Next() {
		var credential model.Credential
		var accessedAtStr string
		if err := rows.Scan(
			&credential.Id,
			&credential.Label,
			&credential.User,
			&credential.AccessCount,
			&credential.Secret,
			&credential.Ephemeral,
			&credential.Nonce,
			&accessedAtStr,
		); err != nil {
			logger.Debug("unable to scan credential")
			return nil, err
		}

		credential.AccessedAt, err = time.Parse(time.RFC3339, accessedAtStr)
		if err != nil {
			logger.Debug("unable to parse time string %s", accessedAtStr)
			return nil, err
		}

		credentials = append(credentials, credential)
	}

	if rows.Err() != nil {
		logger.Debug("error iterating over rows")
		return nil, rows.Err()
	}

	return credentials, nil
}

func UpdateCredentialAccessCount(id, delta int, accessTime time.Time) error {
	query := `UPDATE credentials SET access_count = access_count + ?, accessed_at = ? WHERE id = ?`
	_, err := db.Exec(query, delta, accessTime, id)
	if err != nil {
		logger.Debug("unable to update credential access info : %d at %s", id, accessTime)
	}

	if constants.ACCESS_COUNT_RESET_THRESHOLD > 0 {
		accessCount := 0
		query := `SELECT access_count FROM credentials WHERE id = ?`
		result := db.QueryRow(query, id)
		result.Scan(&accessCount)
		if err := result.Err(); err != nil {
			logger.Debug("unable to get existing access count for id %d", id)
			return err
		}

		if accessCount > constants.ACCESS_COUNT_RESET_THRESHOLD {
			logger.Debug("access count baseline reset triggered")
			// reset access count base-line to prevent a single credentials from dominating
			// the search un-fairly.
			err := ResetAccessCountBaseline()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ResetAccessCountBaseline resets the access count for all credentials by ACCESS_COUNT_RESET_THRESHOLD
// clamping values between [0, ACCESS_COUNT_RESET_THRESHOLD]. Access count less than ACCESS_COUNT_RESET_THRESHOLD is
// set to 0.
func ResetAccessCountBaseline() error {
	query := `UPDATE credentials SET access_count = MAX(access_count - ?, 0)`
	_, err := db.Exec(query, constants.ACCESS_COUNT_RESET_THRESHOLD)
	if err != nil {
		logger.Debug("failed to update access count baseline")
		return err
	}
	return nil
}
