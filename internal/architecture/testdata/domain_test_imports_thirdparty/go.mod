module example.test/archfixture

go 1.25

// A local replace, so go list resolves the fake third-party module offline
// and with no go.sum entry.
require example.test/thirdparty v0.0.0

replace example.test/thirdparty => ./thirdparty
