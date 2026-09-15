package execution

// Test-only, so ADR-0003 permits it: the import lands in go list's
// TestImports and no non-test build compiles this file.
import _ "example.test/thirdparty"
