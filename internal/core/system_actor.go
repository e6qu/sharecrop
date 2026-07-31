package core

// systemUserIDValue is the fixed identity of the durable background actor
// seeded by migration 000039_system_actor.sql. Background sweeps (privacy
// retention, reservation and task expiry) record it as their acting user so
// actor foreign keys and audit trails stay intact. The row has no credentials
// and registration rejects its reserved email, so it can never authenticate.
const systemUserIDValue = "00000000-0000-7000-8000-000000000001"

// SystemUserEmail is the reserved address of the system actor row.
const SystemUserEmail = "system@sharecrop.invalid"

var systemUserID = func() UserID {
	created, matched := ParseUserID(systemUserIDValue).(UserIDCreated)
	if !matched {
		return UserID{}
	}
	return created.Value
}()

// SystemUserID returns the durable system actor identity. A test asserts the
// constant parses, so the zero fallback above is unreachable in practice.
func SystemUserID() UserID {
	return systemUserID
}
