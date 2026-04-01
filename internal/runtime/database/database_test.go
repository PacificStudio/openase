package database

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOpenRejectsEmptyDSN(t *testing.T) {
	t.Helper()

	if _, err := Open(context.Background(), "   "); err == nil || !strings.Contains(err.Error(), "database.dsn is required") {
		t.Fatalf("Open(empty dsn) error = %v", err)
	}
}

func TestOpenFailsForMalformedDSN(t *testing.T) {
	t.Helper()

	if _, err := Open(context.Background(), "postgres://%zz"); err == nil {
		t.Fatal("Open(malformed dsn) expected error")
	}
}

func TestOpenCreatesCurrentSchemaOnFreshDatabase(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	dsn := startEmbeddedPostgres(t)

	client, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close runtime ent client: %v", err)
		}
	})

	org, err := client.Organization.Create().
		SetName("Acme").
		SetSlug("acme").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}

	if _, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Payments").
		SetSlug("payments").
		Save(ctx); err != nil {
		t.Fatalf("create project: %v", err)
	}
}

func TestWithSchemaBootstrapLockReturnsFunctionError(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	dsn := startEmbeddedPostgres(t)
	wantErr := errors.New("boom")

	err := withSchemaBootstrapLock(ctx, dsn, func() error {
		return wantErr
	})
	if err != wantErr {
		t.Fatalf("withSchemaBootstrapLock() error = %v, want %v", err, wantErr)
	}
}

func TestWithSchemaBootstrapLockExecutesCallbackOnSuccess(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	dsn := startEmbeddedPostgres(t)
	called := false

	if err := withSchemaBootstrapLock(ctx, dsn, func() error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("withSchemaBootstrapLock() error = %v", err)
	}
	if !called {
		t.Fatal("withSchemaBootstrapLock() expected callback to run")
	}
}

func TestWithSchemaBootstrapLockRejectsMalformedDSN(t *testing.T) {
	t.Helper()

	if err := withSchemaBootstrapLock(context.Background(), "postgres://%zz", func() error {
		return nil
	}); err == nil {
		t.Fatal("withSchemaBootstrapLock(malformed dsn) expected error")
	}
}

func TestWithSchemaBootstrapLockHonorsCanceledContext(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := withSchemaBootstrapLock(ctx, startEmbeddedPostgres(t), func() error {
		t.Fatal("withSchemaBootstrapLock() should not invoke callback when context is canceled")
		return nil
	})
	if err == nil {
		t.Fatal("withSchemaBootstrapLock(canceled context) expected error")
	}
}

func TestWithSchemaBootstrapLockSerializesConcurrentCallers(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	const waitTimeout = 15 * time.Second

	dsn := startEmbeddedPostgres(t)
	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondEntered := make(chan struct{})
	firstDone := make(chan error, 1)
	secondDone := make(chan error, 1)

	go func() {
		firstDone <- withSchemaBootstrapLock(ctx, dsn, func() error {
			close(firstEntered)
			<-releaseFirst
			return nil
		})
	}()

	select {
	case <-firstEntered:
	case err := <-firstDone:
		t.Fatalf("first schema bootstrap lock caller failed before entering critical section: %v", err)
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for first schema bootstrap lock holder")
	}

	go func() {
		secondDone <- withSchemaBootstrapLock(ctx, dsn, func() error {
			close(secondEntered)
			return nil
		})
	}()

	select {
	case <-secondEntered:
		t.Fatal("expected second schema bootstrap caller to wait for lock release")
	case err := <-secondDone:
		t.Fatalf("second schema bootstrap lock caller finished before lock release: %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	close(releaseFirst)

	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first schema bootstrap lock caller failed: %v", err)
		}
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for first schema bootstrap lock caller to finish")
	}

	select {
	case <-secondEntered:
	case err := <-secondDone:
		t.Fatalf("second schema bootstrap lock caller finished before signalling entry: %v", err)
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for second schema bootstrap lock caller to enter")
	}

	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("second schema bootstrap lock caller failed: %v", err)
		}
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for second schema bootstrap lock caller to finish")
	}
}

func startEmbeddedPostgres(t *testing.T) string {
	t.Helper()

	return testPostgres.NewIsolatedDatabase(t).DSN
}
