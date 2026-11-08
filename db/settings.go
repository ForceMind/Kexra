package db

import "database/sql"

// Settings is a tiny key-value store for global app config the UI toggles at
// runtime (e.g. traffic_capture). Missing keys fall back to caller defaults.

// GetSetting returns the stored value and ok=false when the key is unset.
func (d *DB) GetSetting(key string) (value string, ok bool, err error) {
	err = d.QueryRow(`SELECT value FROM settings WHERE key=$1`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

// SetSetting upserts a setting value.
func (d *DB) SetSetting(key, value string) error {
	_, err := d.Exec(`
INSERT INTO settings(key, value) VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, key, value)
	return err
}

// SetSettingIfEmpty atomically sets an absent or empty setting. Exactly one
// concurrent caller providing a nonempty value can replace the empty value;
// a nonempty setting is retained. Callers claiming initialization must provide
// a nonempty value.
// The bool reports whether this call stored value. Database errors are never
// converted into a successful claim.
func (d *DB) SetSettingIfEmpty(key, value string) (bool, error) {
	result, err := d.Exec(`
INSERT INTO settings(key, value) VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value, updated_at = now()
WHERE settings.value = ''`, key, value)
	if err != nil {
		return false, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return changed == 1, nil
}

// GetBool returns the boolean setting, or def when unset/unparseable.
func (d *DB) GetBool(key string, def bool) bool {
	v, ok, err := d.GetSetting(key)
	if err != nil || !ok {
		return def
	}
	return v == "true" || v == "1"
}

// SetBool stores a boolean setting as "true"/"false".
func (d *DB) SetBool(key string, val bool) error {
	if val {
		return d.SetSetting(key, "true")
	}
	return d.SetSetting(key, "false")
}
