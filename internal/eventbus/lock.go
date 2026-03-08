package eventbus

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (s *Store) ReadLock() (Lease, error) {
	content, err := readMarkdown(s.paths.lock)
	if err != nil {
		return Lease{}, err
	}
	if isPlaceholderDocument(content) {
		return Lease{}, ErrBusNotInitialized
	}
	lease, err := parseLock(content)
	if err != nil {
		return Lease{}, err
	}
	if lease.Status == LockStatusHeld && isExpired(lease) {
		lease.Status = LockStatusExpired
	}
	return lease, nil
}

func (s *Store) AcquireLock(holder string, ttl time.Duration) (Lease, error) {
	current, err := s.ReadLock()
	if err != nil {
		return Lease{}, err
	}
	if current.Status == LockStatusHeld {
		return Lease{}, ErrLeaseConflict
	}
	now := utcNow()
	lease := Lease{
		Status:        LockStatusHeld,
		Holder:        holder,
		LeaseToken:    leaseTokenFunc(),
		LeaseVersion:  current.LeaseVersion + 1,
		AcquiredAt:    now,
		RenewedAt:     now,
		ExpireAt:      nowFunc().UTC().Add(ttl).Format(time.RFC3339),
		LastOperation: "acquire",
	}
	if err := atomicWrite(s.paths.lock, renderLock(lease)); err != nil {
		return Lease{}, err
	}
	return lease, nil
}

func (s *Store) RenewLock(lease Lease, ttl time.Duration) (Lease, error) {
	current, err := s.ReadLock()
	if err != nil {
		return Lease{}, err
	}
	if current.Status == LockStatusExpired {
		return Lease{}, ErrLeaseExpired
	}
	if !sameLease(current, lease) {
		return Lease{}, ErrLeaseConflict
	}
	renewed := current
	renewed.LeaseVersion++
	renewed.RenewedAt = utcNow()
	renewed.ExpireAt = nowFunc().UTC().Add(ttl).Format(time.RFC3339)
	renewed.LastOperation = "renew"
	if err := atomicWrite(s.paths.lock, renderLock(renewed)); err != nil {
		return Lease{}, err
	}
	return renewed, nil
}

func (s *Store) ReleaseLock(lease Lease) error {
	current, err := s.ReadLock()
	if err != nil {
		return err
	}
	if current.Status == LockStatusExpired {
		return ErrLeaseExpired
	}
	if !sameLease(current, lease) {
		return ErrLeaseConflict
	}
	released := Lease{
		Status:        LockStatusFree,
		LeaseVersion:  current.LeaseVersion + 1,
		LastOperation: "release",
	}
	return atomicWrite(s.paths.lock, renderLock(released))
}

func parseLock(content string) (Lease, error) {
	values := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(trimmed, "- "), ":", 2)
		if len(parts) != 2 {
			continue
		}
		values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	version, err := strconv.ParseInt(defaultString(values["lease_version"], "0"), 10, 64)
	if err != nil {
		return Lease{}, ErrBusNotInitialized
	}
	return Lease{
		Status:        defaultString(values["status"], LockStatusFree),
		Holder:        values["holder"],
		LeaseToken:    values["lease_token"],
		LeaseVersion:  version,
		AcquiredAt:    values["acquired_at"],
		RenewedAt:     values["renewed_at"],
		ExpireAt:      values["expire_at"],
		LastOperation: values["last_operation"],
	}, nil
}

func renderLock(lease Lease) []byte {
	lines := []string{
		"# Bus Lock",
		"",
		fmt.Sprintf("- status: %s", defaultString(lease.Status, LockStatusFree)),
		fmt.Sprintf("- holder: %s", lease.Holder),
		fmt.Sprintf("- lease_token: %s", lease.LeaseToken),
		fmt.Sprintf("- lease_version: %d", lease.LeaseVersion),
		fmt.Sprintf("- acquired_at: %s", lease.AcquiredAt),
		fmt.Sprintf("- renewed_at: %s", lease.RenewedAt),
		fmt.Sprintf("- expire_at: %s", lease.ExpireAt),
		fmt.Sprintf("- last_operation: %s", lease.LastOperation),
	}
	return []byte(fmt.Sprintf("%s\n", joinLines(lines)))
}

func sameLease(current Lease, incoming Lease) bool {
	return current.Holder == incoming.Holder && current.LeaseToken == incoming.LeaseToken && current.LeaseVersion == incoming.LeaseVersion
}

func isExpired(lease Lease) bool {
	if lease.ExpireAt == "" {
		return false
	}
	expireAt, err := time.Parse(time.RFC3339, lease.ExpireAt)
	if err != nil {
		return true
	}
	return !nowFunc().UTC().Before(expireAt)
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
