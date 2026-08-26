package store

func idempotencyScope(caseID, operation string) string { return operation + ":" + caseID }
