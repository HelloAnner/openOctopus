package eventbus

import (
	"errors"
	"testing"
	"time"
)

func TestAcquireRenewReleaseLock(t *testing.T) {
	store, _ := bootstrapEventBusStore(t)

	lease, err := store.AcquireLock("orchestrator/master", 30*time.Second)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	if lease.Holder != "orchestrator/master" || lease.LeaseVersion != 1 {
		t.Fatalf("unexpected lease after acquire: %+v", lease)
	}

	renewed, err := store.RenewLock(lease, 30*time.Second)
	if err != nil {
		t.Fatalf("renew lock: %v", err)
	}
	if renewed.LeaseVersion != 2 {
		t.Fatalf("expected renewed version 2, got %+v", renewed)
	}

	err = store.ReleaseLock(lease)
	if !errors.Is(err, ErrLeaseConflict) {
		t.Fatalf("expected ErrLeaseConflict for stale release, got %v", err)
	}

	err = store.ReleaseLock(renewed)
	if err != nil {
		t.Fatalf("release renewed lease: %v", err)
	}

	lockState, err := store.ReadLock()
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if lockState.Status != LockStatusFree || lockState.LeaseVersion != 3 {
		t.Fatalf("unexpected free lock state: %+v", lockState)
	}
}

func TestExpiredLeaseCanNotBeReused(t *testing.T) {
	useFixedEventBusClock(t)
	store, _ := bootstrapEventBusStore(t)

	lease, err := store.AcquireLock("orchestrator/master", time.Second)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	nowFunc = func() time.Time {
		return time.Date(2026, time.March, 8, 12, 0, 2, 0, time.UTC)
	}

	_, err = store.RenewLock(lease, time.Second)
	if !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expected ErrLeaseExpired, got %v", err)
	}

	reacquired, err := store.AcquireLock("human-gate", 30*time.Second)
	if err != nil {
		t.Fatalf("reacquire expired lock: %v", err)
	}
	if reacquired.Holder != "human-gate" || reacquired.LeaseVersion != 2 {
		t.Fatalf("unexpected reacquired lease: %+v", reacquired)
	}
}
