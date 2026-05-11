package postgres

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxInt32Value = int(^uint32(0) >> 1)
	minInt32Value = -maxInt32Value - 1
)

func toPgUUIDPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func toPgTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil || t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromPgUUID(uid pgtype.UUID) uuid.UUID {
	if !uid.Valid {
		return uuid.Nil
	}
	id, _ := uuid.FromBytes(uid.Bytes[:])
	return id
}

func fromPgUUIDPtr(uid pgtype.UUID) *uuid.UUID {
	if !uid.Valid {
		return nil
	}
	id := fromPgUUID(uid)
	return &id
}

func fromPgTimestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func int32PtrOrNil(v int) *int32 {
	if v <= 0 {
		return nil
	}

	v32, err := intToInt32Checked(v)
	if err != nil {
		return nil
	}

	return &v32
}

func intToInt32Checked(v int) (int32, error) {
	if v < minInt32Value || v > maxInt32Value {
		return 0, fmt.Errorf("int value %d overflows int32", v)
	}

	return int32(v), nil
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func int32Value(v *int32) int {
	if v == nil {
		return 0
	}
	return int(*v)
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}

	v := s
	return &v
}

func boolPtr(v bool) *bool {
	b := v
	return &b
}
