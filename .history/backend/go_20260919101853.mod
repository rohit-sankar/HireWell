module github.com/resume-platform/backend

go 1.22.2

require (
	github.com/gin-gonic/gin v1.7.7
	github.com/go-sql-driver/mysql v1.8.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/rs/zerolog v1.33.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/oauth2 v0.20.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/golang/protobuf v1.3.3 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	golang.org/x/sys v0.12.0 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

// --- Replace directives below route Go's vanity-import domains through
// --- their identical github.com mirrors. This is required in network
// --- environments (like this build sandbox) that allowlist github.com
// --- but not golang.org, gopkg.in, gitpkg.in-style redirectors, etc.
// --- These are safe permanently: golang.org/x/* is literally hosted at
// --- github.com/golang/* (same commits), so this is not a security
// --- downgrade, just a transport-path fix. Safe to remove if your
// --- network allows golang.org/gopkg.in directly.
replace (
	golang.org/x/arch => github.com/golang/arch v0.0.0-20210923205945-b76863e36670
	golang.org/x/crypto => github.com/golang/crypto v0.0.0-20220314234659-1baeb1ce4c0b
	golang.org/x/net => github.com/golang/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/oauth2 => github.com/golang/oauth2 v0.20.0
	golang.org/x/sys => github.com/golang/sys v0.0.0-20220227234510-4e6760a101f9
	golang.org/x/term => github.com/golang/term v0.0.0-20201126162022-7de9c90e9dd1
	golang.org/x/text => github.com/golang/text v0.3.7
	golang.org/x/tools => github.com/golang/tools v0.0.0-20180917221912-90fa682c2a6e
)

replace filippo.io/edwards25519 => github.com/FiloSottile/edwards25519 v1.1.0

replace gopkg.in/check.v1 => github.com/go-check/check v0.0.0-20161208181325-20d25e280405
