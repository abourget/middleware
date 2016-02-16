package jwt

import (
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
)

// SigningMethod is the enum that lists the supported token signature hashing algorithms.
type SigningMethod int

const (
	_ = iota
	// RSA256 signing algorithm
	RSA256 SigningMethod = iota + 1
	// RSA384 signing algorithm
	RSA384
	// RSA512 signing algorithm
	RSA512
	// HMAC256 signing algorithm
	HMAC256
	// HMAC384 signing algorithm
	HMAC384
	// HMAC512 signing algorithm
	HMAC512
	// ECDSA256 signing algorithm
	ECDSA256
	// ECDSA384 signing algorithm
	ECDSA384
	// ECDSA512 signing algorithm
	ECDSA512
)

var signingmethods map[SigningMethod]string

func init() {
	signingmethods = make(map[SigningMethod]string)

	signingmethods[RSA256] = "RS256"
	signingmethods[RSA384] = "RS512"
	signingmethods[RSA512] = "RS512"
	signingmethods[HMAC256] = "HS256"
	signingmethods[HMAC384] = "HS384"
	signingmethods[HMAC512] = "HS512"
	signingmethods[ECDSA256] = "ES256"
	signingmethods[ECDSA384] = "ES384"
	signingmethods[ECDSA512] = "ES512"
}

// JWTKey is the JWT middleware key used to store the token in the context.
const JWTKey middlewareKey = 0

// TokenManagerKey is the JWT middleware key used to store the token manager in the context.
const TokenManagerKey middlewareKey = 1

// JWTHeader is the name of the header used to transmit the JWT token.
const JWTHeader = "Authorization"

// ValidationKeyfunc is a function that takes a token and returns the key to validate that
// token, which allows it to use inforamtion from the key itself to choose the key
// to return.
type ValidationKeyfunc func(*jwt.Token) (interface{}, error)

func keyFuncWrapper(k ValidationKeyfunc) jwt.Keyfunc {
	return func(tok *jwt.Token) (interface{}, error) {
		return k(tok)
	}
}

// KeyFunc is a function that returns the key to sign a
// token.  It should return a []byte (for all)
// or a *rsa.PrivateKey or *ecdsa.PrivateKey
type KeyFunc func() (interface{}, error)

// Specification describes the JWT authorization properties.
// It is used to both instantiate a middleware and a token manager.
// The middleware uses the specification properties to authorize the incoming
// request while the token manager uses it to create authorization tokens.
type Specification struct {
	// TokenHeader is the HTTP header to search for the JWT Token
	// Defaults to "Authorization"
	TokenHeader string
	// TokenParam is the request parameter to parse for the JWT Token
	// Defaults to "token"
	TokenParam string
	// AllowParam is a flag that determines whether it is allowable
	// to parse tokens from the querystring
	// Defaults to false
	AllowParam bool
	// ValidationFunc is a function that returns the key to validate the JWT
	// Required, no default
	ValidationFunc ValidationKeyfunc
	// AuthOptions is a flag that determines whether a token is required on OPTIONS
	// requests
	AuthOptions bool
	// TTLMinutes is the TTL for tokens that are generated
	TTLMinutes int
	// RefreshTTLMinutes is the TTL for refresh tokens that are generated
	// and should generally be much longer than TTLMinutes
	RefreshTTLMinutes int
	// Issuer is the name of the issuer that will be inserted into the
	// generated token's claims
	Issuer string
	// KeySigningMethod determines the type of key that will be used to sign
	// Tokens.
	KeySigningMethod SigningMethod
	// SigningKeyFunc is a function that returns the key used to sign the token
	SigningKeyFunc KeyFunc
	// CommonClaims is a list of claims added to all tokens issued
	CommonClaims map[string]interface{}
}

// GetToken extracts the JWT token from the request if there is one.
func GetToken(req *http.Request, spec *Specification) (token *jwt.Token, err error) {
	var found bool
	var tok string
	header := req.Header.Get(spec.TokenHeader)

	if header != "" {
		parts := strings.Split(header, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			err = fmt.Errorf("Malformed token header")
			return
		}
		tok = parts[1]
		found = true
	}
	if !found && spec.AllowParam {
		tok = req.URL.Query().Get(spec.TokenParam)
	}
	if tok == "" {
		err = fmt.Errorf("no token")
		return
	}
	token, err = jwt.Parse(tok, keyFuncWrapper(spec.ValidationFunc))
	return
}

// Middleware is a middleware that retrieves a JWT token from the request if present and
// injects it into the context.  It checks for the token in the HTTP Headers first, then the querystring if
// the specification "AllowParam" is true.
// Retrieve it using ctx.Value(JWTKey).
func Middleware(spec *Specification) goa.Middleware {
	if spec.TokenHeader == "" {
		spec.TokenHeader = "Authorization"
	}
	if spec.TokenParam == "" {
		spec.TokenParam = "token"
	}
	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			// If AuthOptions is false, and this is an OPTIONS request
			// just let the request fly
			if !spec.AuthOptions && req.Method == "OPTIONS" {
				return h(ctx, rw, req)
			}
			token, err := GetToken(req, spec)
			if err != nil {
				return goa.Response(ctx).Send(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			}
			if !token.Valid {
				return goa.Response(ctx).Send(http.StatusUnauthorized, "Invalid Token")
			}

			ctx = context.WithValue(ctx, JWTKey, token)
			return h(ctx, rw, req)
		}

	}
}
