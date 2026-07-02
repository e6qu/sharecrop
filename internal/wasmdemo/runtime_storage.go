package wasmdemo

import (
	"encoding/json"
	"errors"
	"strings"
)

func putEncodedRecord(storage BrowserStorage, key string, encoded string, label string) error {
	keyResult := NewStorageKey(key)
	accepted, matched := keyResult.(StorageKeyAccepted)
	if !matched {
		return errors.New(keyResult.(StorageKeyRejected).Reason)
	}
	writeResult := storage.Put(accepted.Value, encoded)
	if _, matched := writeResult.(StorageWritten); !matched {
		return errors.New(writeResult.(StorageWriteRejected).Reason)
	}
	return nil
}

func readRecordString(storage BrowserStorage, key string, label string) (string, error) {
	keyResult := NewStorageKey(key)
	accepted, matched := keyResult.(StorageKeyAccepted)
	if !matched {
		return "", errors.New(keyResult.(StorageKeyRejected).Reason)
	}
	readResult := storage.Get(accepted.Value)
	read, matched := readResult.(StorageRead)
	if !matched {
		return "", errors.New(storageReadReason(readResult, label))
	}
	return read.Value, nil
}

func SaveUser(storage BrowserStorage, user StoredUser) error {
	user.ID = strings.TrimSpace(user.ID)
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	user.Status = strings.TrimSpace(user.Status)
	if user.ID == "" || user.Email == "" {
		return errors.New("user id and email are required")
	}
	if user.Status == "" {
		user.Status = "active"
	}
	encoded, err := json.Marshal(user)
	if err != nil {
		return errors.New("user encoding failed")
	}
	if err := putEncodedRecord(storage, "user:"+user.ID, string(encoded), "user"); err != nil {
		return err
	}
	encodedEmail, err := json.Marshal(user.ID)
	if err != nil {
		return errors.New("user email encoding failed")
	}
	if err := putEncodedRecord(storage, "user_email:"+user.Email, string(encodedEmail), "user email"); err != nil {
		return err
	}
	if result := appendStringIndex(storage, "user:index", user.ID, "user"); result != (stringIndexStored{}) {
		return errors.New(result.(stringIndexRejected).reason)
	}
	return nil
}

func LoadUser(storage BrowserStorage, userID string) (StoredUser, error) {
	var user StoredUser
	raw, err := readRecordString(storage, "user:"+strings.TrimSpace(userID), "user")
	if err != nil {
		return user, err
	}
	if err := json.Unmarshal([]byte(raw), &user); err != nil {
		return user, errors.New("user decoding failed")
	}
	return user, nil
}

func LoadUserIDByEmail(storage BrowserStorage, email string) (string, error) {
	var userID string
	raw, err := readRecordString(storage, "user_email:"+strings.ToLower(strings.TrimSpace(email)), "user email")
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal([]byte(raw), &userID); err != nil {
		return "", errors.New("user email decoding failed")
	}
	return strings.TrimSpace(userID), nil
}

func ListUsers(storage BrowserStorage, query string, page StoredListPage) ([]StoredUser, error) {
	idsResult := loadStringIndex(storage, "user:index", "user")
	ids, matched := idsResult.(stringIndexLoaded)
	if !matched {
		return nil, errors.New(idsResult.(stringIndexRejected).reason)
	}
	cleanQuery := strings.ToLower(strings.TrimSpace(query))
	values := make([]StoredUser, 0, len(ids.values))
	for index := range ids.values {
		user, err := LoadUser(storage, ids.values[index])
		if err != nil {
			return nil, err
		}
		if cleanQuery == "" || strings.Contains(strings.ToLower(user.Email), cleanQuery) || strings.Contains(strings.ToLower(user.ID), cleanQuery) {
			values = append(values, user)
		}
	}
	start, end := pageBounds(len(values), page)
	return values[start:end], nil
}

func SaveAuditEvent(storage BrowserStorage, event StoredAuditEvent) error {
	event.ID = strings.TrimSpace(event.ID)
	event.ActorID = strings.TrimSpace(event.ActorID)
	event.Action = strings.TrimSpace(event.Action)
	event.SubjectKind = strings.TrimSpace(event.SubjectKind)
	event.SubjectID = strings.TrimSpace(event.SubjectID)
	event.CreatedAt = strings.TrimSpace(event.CreatedAt)
	if event.ID == "" || event.ActorID == "" || event.Action == "" || event.SubjectKind == "" || event.SubjectID == "" || event.CreatedAt == "" {
		return errors.New("audit event fields are required")
	}
	if event.MetadataJSON == "" {
		event.MetadataJSON = "{}"
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		return errors.New("audit event encoding failed")
	}
	if err := putEncodedRecord(storage, "audit_event:"+event.ID, string(encoded), "audit event"); err != nil {
		return err
	}
	if result := appendStringIndex(storage, "audit_event:index", event.ID, "audit event"); result != (stringIndexStored{}) {
		return errors.New(result.(stringIndexRejected).reason)
	}
	return nil
}

func ListAuditEvents(storage BrowserStorage, action string, subjectKind string, subjectID string, page StoredListPage) ([]StoredAuditEvent, error) {
	idsResult := loadStringIndex(storage, "audit_event:index", "audit event")
	ids, matched := idsResult.(stringIndexLoaded)
	if !matched {
		return nil, errors.New(idsResult.(stringIndexRejected).reason)
	}
	values := make([]StoredAuditEvent, 0, len(ids.values))
	for index := range ids.values {
		var event StoredAuditEvent
		raw, err := readRecordString(storage, "audit_event:"+ids.values[index], "audit event")
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			return nil, errors.New("audit event decoding failed")
		}
		if strings.TrimSpace(action) != "" && event.Action != strings.TrimSpace(action) {
			continue
		}
		if strings.TrimSpace(subjectKind) != "" && event.SubjectKind != strings.TrimSpace(subjectKind) {
			continue
		}
		if strings.TrimSpace(subjectID) != "" && event.SubjectID != strings.TrimSpace(subjectID) {
			continue
		}
		values = append(values, event)
	}
	start, end := pageBounds(len(values), page)
	return values[start:end], nil
}

func SavePlatformAdmin(storage BrowserStorage, admin StoredPlatformAdmin) error {
	admin.UserID = strings.TrimSpace(admin.UserID)
	admin.Source = strings.TrimSpace(admin.Source)
	admin.State = strings.TrimSpace(admin.State)
	admin.CreatedAt = strings.TrimSpace(admin.CreatedAt)
	if admin.UserID == "" || admin.Source == "" || admin.State == "" || admin.CreatedAt == "" {
		return errors.New("platform admin fields are required")
	}
	encoded, err := json.Marshal(admin)
	if err != nil {
		return errors.New("platform admin encoding failed")
	}
	if err := putEncodedRecord(storage, "platform_admin:"+admin.UserID, string(encoded), "platform admin"); err != nil {
		return err
	}
	if result := appendStringIndex(storage, "platform_admin:index", admin.UserID, "platform admin"); result != (stringIndexStored{}) {
		return errors.New(result.(stringIndexRejected).reason)
	}
	return nil
}

func LoadPlatformAdmin(storage BrowserStorage, userID string) (StoredPlatformAdmin, error) {
	var admin StoredPlatformAdmin
	raw, err := readRecordString(storage, "platform_admin:"+strings.TrimSpace(userID), "platform admin")
	if err != nil {
		return admin, err
	}
	if err := json.Unmarshal([]byte(raw), &admin); err != nil {
		return admin, errors.New("platform admin decoding failed")
	}
	return admin, nil
}

func ListPlatformAdmins(storage BrowserStorage, page StoredListPage) ([]StoredPlatformAdmin, error) {
	idsResult := loadStringIndex(storage, "platform_admin:index", "platform admin")
	ids, matched := idsResult.(stringIndexLoaded)
	if !matched {
		return nil, errors.New(idsResult.(stringIndexRejected).reason)
	}
	values := make([]StoredPlatformAdmin, 0, len(ids.values))
	for index := range ids.values {
		admin, err := LoadPlatformAdmin(storage, ids.values[index])
		if err != nil {
			return nil, err
		}
		if admin.State == "active" {
			values = append(values, admin)
		}
	}
	start, end := pageBounds(len(values), page)
	return values[start:end], nil
}

func SaveAccountToken(storage BrowserStorage, token StoredAccountToken) error {
	token.Token = strings.TrimSpace(token.Token)
	token.Kind = strings.TrimSpace(token.Kind)
	token.UserID = strings.TrimSpace(token.UserID)
	token.State = strings.TrimSpace(token.State)
	if token.Token == "" || token.Kind == "" || token.UserID == "" || token.State == "" {
		return errors.New("account token fields are required")
	}
	encoded, err := json.Marshal(token)
	if err != nil {
		return errors.New("account token encoding failed")
	}
	return putEncodedRecord(storage, "account_token:"+token.Token, string(encoded), "account token")
}

func ConsumeAccountToken(storage BrowserStorage, rawToken string, kind string) error {
	var token StoredAccountToken
	raw, err := readRecordString(storage, "account_token:"+strings.TrimSpace(rawToken), "account token")
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(raw), &token); err != nil {
		return errors.New("account token decoding failed")
	}
	if token.Kind != strings.TrimSpace(kind) || token.State != "active" {
		return errors.New("account token is invalid")
	}
	token.State = "used"
	return SaveAccountToken(storage, token)
}

func SaveCollectible(storage BrowserStorage, collectible StoredCollectible) error {
	collectible.ID = strings.TrimSpace(collectible.ID)
	collectible.Name = strings.TrimSpace(collectible.Name)
	collectible.Kind = strings.TrimSpace(collectible.Kind)
	collectible.State = strings.TrimSpace(collectible.State)
	collectible.TransferPolicy = strings.TrimSpace(collectible.TransferPolicy)
	collectible.OwnerID = strings.TrimSpace(collectible.OwnerID)
	collectible.OwnerKind = strings.TrimSpace(collectible.OwnerKind)
	collectible.OrganizationID = strings.TrimSpace(collectible.OrganizationID)
	collectible.Art = strings.TrimSpace(collectible.Art)
	if collectible.ID == "" || collectible.Name == "" || collectible.Kind == "" || collectible.State == "" || collectible.TransferPolicy == "" || collectible.OwnerID == "" || collectible.OwnerKind == "" {
		return errors.New("collectible fields are required")
	}
	encoded, err := json.Marshal(collectible)
	if err != nil {
		return errors.New("collectible encoding failed")
	}
	if err := putEncodedRecord(storage, "collectible:"+collectible.ID, string(encoded), "collectible"); err != nil {
		return err
	}
	if result := appendStringIndex(storage, "collectible:index", collectible.ID, "collectible"); result != (stringIndexStored{}) {
		return errors.New(result.(stringIndexRejected).reason)
	}
	return nil
}

func LoadCollectible(storage BrowserStorage, collectibleID string) (StoredCollectible, error) {
	var collectible StoredCollectible
	raw, err := readRecordString(storage, "collectible:"+strings.TrimSpace(collectibleID), "collectible")
	if err != nil {
		return collectible, err
	}
	if err := json.Unmarshal([]byte(raw), &collectible); err != nil {
		return collectible, errors.New("collectible decoding failed")
	}
	return collectible, nil
}

func ListCollectibles(storage BrowserStorage, ownerKind string, ownerID string, page StoredListPage) ([]StoredCollectible, error) {
	idsResult := loadStringIndex(storage, "collectible:index", "collectible")
	ids, matched := idsResult.(stringIndexLoaded)
	if !matched {
		return nil, errors.New(idsResult.(stringIndexRejected).reason)
	}
	values := make([]StoredCollectible, 0, len(ids.values))
	for index := range ids.values {
		collectible, err := LoadCollectible(storage, ids.values[index])
		if err != nil {
			return nil, err
		}
		if collectible.OwnerKind == strings.TrimSpace(ownerKind) && collectible.OwnerID == strings.TrimSpace(ownerID) {
			values = append(values, collectible)
		}
	}
	start, end := pageBounds(len(values), page)
	return values[start:end], nil
}

func SaveAgentCredential(storage BrowserStorage, credential StoredAgentCredential) error {
	credential.ID = strings.TrimSpace(credential.ID)
	credential.OwnerID = strings.TrimSpace(credential.OwnerID)
	credential.Label = strings.TrimSpace(credential.Label)
	credential.State = strings.TrimSpace(credential.State)
	if credential.ID == "" || credential.OwnerID == "" || credential.Label == "" || credential.State == "" {
		return errors.New("agent credential fields are required")
	}
	if len(credential.Scopes) == 0 {
		return errors.New("agent credential scopes are required")
	}
	encoded, err := json.Marshal(credential)
	if err != nil {
		return errors.New("agent credential encoding failed")
	}
	if err := putEncodedRecord(storage, "agent_credential:"+credential.ID, string(encoded), "agent credential"); err != nil {
		return err
	}
	if result := appendStringIndex(storage, "agent_credential:index:"+credential.OwnerID, credential.ID, "agent credential"); result != (stringIndexStored{}) {
		return errors.New(result.(stringIndexRejected).reason)
	}
	return nil
}

func LoadAgentCredential(storage BrowserStorage, credentialID string) (StoredAgentCredential, error) {
	var credential StoredAgentCredential
	raw, err := readRecordString(storage, "agent_credential:"+strings.TrimSpace(credentialID), "agent credential")
	if err != nil {
		return credential, err
	}
	if err := json.Unmarshal([]byte(raw), &credential); err != nil {
		return credential, errors.New("agent credential decoding failed")
	}
	return credential, nil
}

func ListAgentCredentials(storage BrowserStorage, ownerID string, page StoredListPage) ([]StoredAgentCredential, error) {
	cleanOwnerID := strings.TrimSpace(ownerID)
	idsResult := loadStringIndex(storage, "agent_credential:index:"+cleanOwnerID, "agent credential")
	ids, matched := idsResult.(stringIndexLoaded)
	if !matched {
		return nil, errors.New(idsResult.(stringIndexRejected).reason)
	}
	values := make([]StoredAgentCredential, 0, len(ids.values))
	for index := range ids.values {
		credential, err := LoadAgentCredential(storage, ids.values[index])
		if err != nil {
			return nil, err
		}
		if credential.OwnerID == cleanOwnerID {
			values = append(values, credential)
		}
	}
	start, end := pageBounds(len(values), page)
	return values[start:end], nil
}
