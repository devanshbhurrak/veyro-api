package identity

import "backend/internal/shared/idgen"

func newUUID() (string, error) { return idgen.New() }
